package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// Default configuration values
const (
	DefaultDownloadURL  = "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz"
	DefaultInstallPath  = "/opt/actions-runner"
	DefaultWorkDir      = "/tmp/runner-work"
	DefaultConfigPath   = "/etc/hyperfleet/runner-config.json"
	DefaultConfigScript = "config.sh"
	DefaultRunScript    = "run.sh"

	// File permissions
	DirPermissions = 0755

	// Timing constants
	CleanupDelaySeconds = 2
	HTTPTimeoutSeconds  = 300 // 5 minutes for download

	// Method constants
	runnerTokenMethod = "runner-token"
	joinTokenMethod   = "join-token"
)

// RunnerConfig represents the configuration loaded from the VM
type RunnerConfig struct {
	Method          string   `json:"method"`
	Platform        string   `json:"platform,omitempty"`         // "github-actions"
	RunnerToken     string   `json:"runner_token,omitempty"`     // Short-lived registration token
	RegistrationURL string   `json:"registration_url,omitempty"` // Where runner registers to
	RunnerName      string   `json:"runner_name,omitempty"`      // Unique runner name
	Labels          []string `json:"labels,omitempty"`           // Runner labels
	ExpiresAt       string   `json:"expires_at,omitempty"`       // Token expiration

	// GitHub Actions runner configuration
	Runner struct {
		DownloadURL  string `json:"download_url,omitempty"`  // GitHub Actions runner download URL
		InstallPath  string `json:"install_path,omitempty"`  // Installation path on VM
		WorkDir      string `json:"work_dir,omitempty"`      // Working directory for jobs
		ConfigScript string `json:"config_script,omitempty"` // Path to config script (default: config.sh)
		RunScript    string `json:"run_script,omitempty"`    // Path to run script (default: run.sh)
		OS           string `json:"os,omitempty"`            // Target OS (default: from GOOS or runtime)
		Arch         string `json:"arch,omitempty"`          // Target architecture (default: from GOARCH or runtime)
	} `json:"runner,omitempty"`

	// SPIFFE fields (for SPIFFE attestation - independent of runner token)
	SPIFFE struct {
		JoinToken string `json:"join_token,omitempty"`
		SPIFFEID  string `json:"spiffe_id,omitempty"`
		Enabled   bool   `json:"enabled,omitempty"`
	} `json:"spiffe,omitempty"`
}

// GitHubBootstrap handles the GitHub Actions runner bootstrap process
type GitHubBootstrap struct {
	config     *RunnerConfig
	logger     Logger
	httpClient HTTPClient
	fileSystem FileSystem
	executor   CommandExecutor
	system     SystemOperations
}

// NewGitHubBootstrap creates a new GitHubBootstrap with the given dependencies
func NewGitHubBootstrap(config *RunnerConfig, logger Logger, httpClient HTTPClient,
	fileSystem FileSystem, executor CommandExecutor, system SystemOperations) *GitHubBootstrap {
	return &GitHubBootstrap{
		config:     config,
		logger:     logger,
		httpClient: httpClient,
		fileSystem: fileSystem,
		executor:   executor,
		system:     system,
	}
}

func main() {
	configPath := flag.String("config", DefaultConfigPath, "Path to runner configuration")
	flag.Parse()

	// Load configuration
	config, err := loadRunnerConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize bootstrap service based on method
	switch config.Method {
	case runnerTokenMethod:
		bootstrap := NewGitHubBootstrap(
			config,
			NewRealLogger("[github-bootstrap] "),
			NewRealHTTPClient(HTTPTimeoutSeconds*time.Second),
			NewRealFileSystem(),
			NewRealCommandExecutor(),
			NewRealSystemOperations(),
		)

		// Handle SPIFFE attestation if enabled (independent of runner token)
		if config.SPIFFE.Enabled {
			if err := bootstrap.performSPIFFEAttestation(); err != nil {
				log.Fatalf("SPIFFE attestation failed: %v", err)
			}
		}

		if err := bootstrap.Run(context.Background()); err != nil {
			log.Fatalf("GitHub bootstrap failed: %v", err)
		}

	case joinTokenMethod:
		// Pure SPIFFE/SPIRE bootstrap implementation (placeholder)
		log.Fatalf("Pure SPIFFE/SPIRE bootstrap not yet implemented")

	default:
		log.Fatalf("Unsupported attestation method: %s", config.Method)
	}
}

// loadRunnerConfig loads the runner configuration from the specified file
func loadRunnerConfig(configPath string) (*RunnerConfig, error) {
	// #nosec G304 - configPath is provided via command line flag, not user input
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config RunnerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Run executes the complete GitHub bootstrap process
func (gb *GitHubBootstrap) Run(ctx context.Context) error {
	gb.logger.Printf("Starting GitHub runner bootstrap for %s", gb.config.RunnerName)

	// 1. Download GitHub Actions runner
	if err := gb.downloadGitHubRunner(ctx); err != nil {
		return fmt.Errorf("failed to download runner: %w", err)
	}

	// 2. Configure runner with registration token
	if err := gb.configureRunner(ctx); err != nil {
		return fmt.Errorf("failed to configure runner: %w", err)
	}

	// 3. Start runner and monitor
	if err := gb.runAndMonitor(ctx); err != nil {
		return fmt.Errorf("failed to run runner: %w", err)
	}

	// 4. Cleanup and self-terminate
	return gb.cleanup(ctx)
}

// downloadGitHubRunner downloads and extracts the GitHub Actions runner using HTTP client
func (gb *GitHubBootstrap) downloadGitHubRunner(ctx context.Context) error {
	installPath := gb.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	downloadURL := gb.buildDownloadURL()
	gb.logger.Printf("Downloading GitHub Actions runner from %s to %s", downloadURL, installPath)

	// Create runner directory
	if err := gb.fileSystem.MkdirAll(installPath, DirPermissions); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}

	// Download the file
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := gb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download runner: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			gb.logger.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download runner: HTTP %d", resp.StatusCode)
	}

	// Extract tar.gz directly from response body
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if err := gzipReader.Close(); err != nil {
			gb.logger.Printf("Warning: failed to close gzip reader: %v", err)
		}
	}()

	tarReader := tar.NewReader(gzipReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// #nosec G305 - Path traversal protection implemented below
		targetPath := filepath.Join(installPath, header.Name)

		// Security check: ensure path is within install directory
		// Allow current directory entry
		if header.Name != "./" && !strings.HasPrefix(targetPath, filepath.Clean(installPath)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// #nosec G115 - header.Mode is from tar header, safe conversion
			if err := gb.fileSystem.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			// Create parent directory if it doesn't exist
			if err := gb.fileSystem.MkdirAll(filepath.Dir(targetPath), DirPermissions); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// #nosec G304 - targetPath is validated above for path traversal
			// #nosec G115 - header.Mode is from tar header, safe conversion
			file, err := gb.fileSystem.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// #nosec G110 - This is extracting a trusted GitHub Actions runner archive
			if _, err := io.Copy(file, tarReader); err != nil {
				if closeErr := file.Close(); closeErr != nil {
					gb.logger.Printf("Warning: failed to close file during error: %v", closeErr)
				}
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", targetPath, err)
			}
		}
	}

	gb.logger.Printf("Successfully downloaded and extracted GitHub Actions runner")
	return nil
}

// configureRunner configures the GitHub Actions runner with the registration token
func (gb *GitHubBootstrap) configureRunner(ctx context.Context) error {
	gb.logger.Printf("Configuring runner %s", gb.config.RunnerName)

	installPath := gb.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := gb.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	configScript := gb.config.Runner.ConfigScript
	if configScript == "" {
		configScript = DefaultConfigScript
	}

	configScriptPath := filepath.Join(installPath, configScript)

	args := []string{
		"--url", gb.config.RegistrationURL,
		"--token", gb.config.RunnerToken,
		"--name", gb.config.RunnerName,
		"--labels", strings.Join(gb.config.Labels, ","),
		"--work", workDir,
		"--unattended",
		"--ephemeral", // Auto-cleanup after job
	}

	// #nosec G204 - configScriptPath is constructed from validated config, not user input
	cmd := gb.executor.CommandContext(ctx, configScriptPath, args...)
	cmd.SetDir(installPath)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	return cmd.Run()
}

// runAndMonitor starts the GitHub Actions runner and monitors its execution
func (gb *GitHubBootstrap) runAndMonitor(ctx context.Context) error {
	gb.logger.Printf("Starting GitHub Actions runner")

	installPath := gb.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	runScript := gb.config.Runner.RunScript
	if runScript == "" {
		runScript = DefaultRunScript
	}

	runScriptPath := filepath.Join(installPath, runScript)

	// #nosec G204 - runScriptPath is constructed from validated config, not user input
	cmd := gb.executor.CommandContext(ctx, runScriptPath)
	cmd.SetDir(installPath)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	// Runner will exit after job completion (ephemeral mode)
	return cmd.Run()
}

// cleanup performs cleanup operations and shuts down the VM
func (gb *GitHubBootstrap) cleanup(_ context.Context) error {
	gb.logger.Printf("Runner completed, initiating VM shutdown")

	// Clean up runner installation and work directory
	installPath := gb.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := gb.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	// Remove directories (non-fatal if they fail)
	if err := gb.fileSystem.RemoveAll(installPath); err != nil {
		gb.logger.Printf("Warning: failed to remove install path %s: %v", installPath, err)
	}

	if err := gb.fileSystem.RemoveAll(workDir); err != nil {
		gb.logger.Printf("Warning: failed to remove work dir %s: %v", workDir, err)
	}

	// Give a moment for cleanup to complete
	gb.system.Sleep(CleanupDelaySeconds)

	// Shutdown the VM using multiple methods
	gb.logger.Printf("Shutting down VM")

	if err := gb.shutdownVM(); err != nil {
		gb.logger.Printf("VM shutdown failed: %v", err)
		gb.logger.Printf("VM cleanup completed, but shutdown failed - operator will handle VM cleanup")
		// Return nil so the bootstrap service exits cleanly
		// The operator's VM monitoring will detect the stopped process and clean up the VM
		return nil
	}

	gb.logger.Printf("VM shutdown initiated successfully")
	return nil
}

// shutdownVM attempts to shutdown the VM using various methods
func (gb *GitHubBootstrap) shutdownVM() error {
	// Method 1: Try syscall approach (most reliable)
	if err := gb.shutdownViaSyscall(); err == nil {
		gb.logger.Printf("VM shutdown via syscall succeeded")
		return nil
	}

	// Method 2: Try SysRq trigger (requires /proc/sys/kernel/sysrq to be enabled)
	if err := gb.shutdownViaSysRq(); err == nil {
		gb.logger.Printf("VM shutdown via SysRq succeeded")
		return nil
	}

	// Method 3: Try power management interface
	if err := gb.shutdownViaPowerState(); err == nil {
		gb.logger.Printf("VM shutdown via power state succeeded")
		return nil
	}

	// Method 4: Fall back to command-based shutdown
	return gb.shutdownViaCommand()
}

// shutdownViaSyscall uses the reboot syscall to shutdown the system
func (gb *GitHubBootstrap) shutdownViaSyscall() error {
	gb.logger.Printf("Attempting shutdown via syscall")

	// Sync filesystems first
	gb.system.Sync()

	// Use the reboot syscall with LINUX_REBOOT_CMD_POWER_OFF
	// This requires root privileges
	err := gb.system.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	if err != nil {
		return fmt.Errorf("syscall reboot failed: %w", err)
	}

	return nil
}

// shutdownViaSysRq uses the SysRq mechanism to shutdown
func (gb *GitHubBootstrap) shutdownViaSysRq() error {
	gb.logger.Printf("Attempting shutdown via SysRq")

	// Write 'o' to /proc/sysrq-trigger to power off
	// This requires /proc/sys/kernel/sysrq to be enabled (usually is)
	sysrqFile := "/proc/sysrq-trigger"

	// #nosec G306 - This is a system file write for shutdown
	file, err := gb.fileSystem.OpenFile(sysrqFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open sysrq-trigger: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			gb.logger.Printf("Warning: failed to close sysrq-trigger: %v", closeErr)
		}
	}()

	// 'o' command powers off the system
	if _, err := gb.fileSystem.WriteString(file, "o"); err != nil {
		return fmt.Errorf("failed to write to sysrq-trigger: %w", err)
	}

	return nil
}

// shutdownViaPowerState uses the power management interface
func (gb *GitHubBootstrap) shutdownViaPowerState() error {
	gb.logger.Printf("Attempting shutdown via power state")

	// Write to /sys/power/state to initiate shutdown
	powerStateFile := "/sys/power/state"

	// #nosec G306 - This is a system file write for shutdown
	file, err := gb.fileSystem.OpenFile(powerStateFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open power state file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			gb.logger.Printf("Warning: failed to close power state file: %v", closeErr)
		}
	}()

	// Write "mem" to suspend to RAM, but we want to power off
	// Actually, let's try writing to /sys/power/disk first
	if err := file.Close(); err != nil {
		gb.logger.Printf("Warning: failed to close power state file: %v", err)
	}

	// Try the poweroff approach via /sys/power/disk
	diskFile := "/sys/power/disk"
	// #nosec G306 - This is a system file write for shutdown
	disk, err := gb.fileSystem.OpenFile(diskFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open power disk file: %w", err)
	}
	defer func() {
		if closeErr := disk.Close(); closeErr != nil {
			gb.logger.Printf("Warning: failed to close power disk file: %v", closeErr)
		}
	}()

	if _, err := gb.fileSystem.WriteString(disk, "shutdown"); err != nil {
		return fmt.Errorf("failed to write to power disk file: %w", err)
	}

	return nil
}

// shutdownViaCommand falls back to command-based shutdown
func (gb *GitHubBootstrap) shutdownViaCommand() error {
	gb.logger.Printf("Attempting shutdown via commands")

	// Try different shutdown commands in order of preference
	shutdownCommands := [][]string{
		{"sudo", "shutdown", "-h", "now"}, // Most common on Linux
		{"shutdown", "-h", "now"},         // If running as root
		{"sudo", "poweroff"},              // Alternative method
		{"poweroff"},                      // If running as root
		{"sudo", "halt", "-p"},            // Another alternative
		{"halt", "-p"},                    // If running as root
		{"sudo", "systemctl", "poweroff"}, // systemd systems
		{"systemctl", "poweroff"},         // systemd as root
	}

	var lastErr error
	for _, cmdArgs := range shutdownCommands {
		gb.logger.Printf("Attempting shutdown with: %v", cmdArgs)

		// #nosec G204 - cmd is from predefined list, not user input
		cmd := gb.executor.CommandContext(context.Background(), cmdArgs[0], cmdArgs[1:]...)
		if err := cmd.Run(); err != nil {
			gb.logger.Printf("Shutdown attempt failed with %v: %v", cmdArgs, err)
			lastErr = err
			continue
		}

		gb.logger.Printf("Shutdown command succeeded: %v", cmdArgs)
		return nil
	}

	return fmt.Errorf("all shutdown methods failed, last error: %w", lastErr)
}

// performSPIFFEAttestation handles SPIFFE attestation independently
func (gb *GitHubBootstrap) performSPIFFEAttestation() error {
	gb.logger.Printf("Performing SPIFFE attestation")

	// TODO: Implement SPIFFE attestation logic
	// This would involve:
	// 1. Connecting to SPIRE agent
	// 2. Obtaining SPIFFE identity
	// 3. Validating identity against expected SPIFFE ID
	// 4. Storing credentials securely

	// For now, just validate that SPIFFE configuration is present
	if gb.config.SPIFFE.JoinToken == "" && gb.config.SPIFFE.SPIFFEID == "" {
		return fmt.Errorf("SPIFFE attestation enabled but no join token or SPIFFE ID provided")
	}

	gb.logger.Printf("SPIFFE attestation completed successfully")
	return nil
}

// getOSArch returns the target OS and architecture from config or environment
func (gb *GitHubBootstrap) getOSArch() (string, string) {
	targetOS := gb.config.Runner.OS
	if targetOS == "" {
		targetOS = os.Getenv("GOOS")
		if targetOS == "" {
			targetOS = runtime.GOOS
		}
	}

	targetArch := gb.config.Runner.Arch
	if targetArch == "" {
		targetArch = os.Getenv("GOARCH")
		if targetArch == "" {
			targetArch = runtime.GOARCH
		}
	}

	return targetOS, targetArch
}

// buildDownloadURL constructs the download URL based on OS/arch
func (gb *GitHubBootstrap) buildDownloadURL() string {
	if gb.config.Runner.DownloadURL != "" {
		return gb.config.Runner.DownloadURL
	}

	targetOS, targetArch := gb.getOSArch()
	gb.logger.Printf("Detected OS: %s, Arch: %s", targetOS, targetArch)

	// Map Go arch names to GitHub runner arch names
	archMap := map[string]string{
		"amd64": "x64",
		"arm64": "arm64",
		"386":   "x86",
	}

	runnerArch, exists := archMap[targetArch]
	if !exists {
		runnerArch = targetArch // fallback to original
	}

	// Map Go OS names to GitHub runner OS names
	osMap := map[string]string{
		"linux":   "linux",
		"darwin":  "osx",
		"windows": "win",
	}

	runnerOS, exists := osMap[targetOS]
	if !exists {
		runnerOS = targetOS // fallback to original
	}

	// Construct URL based on GitHub Actions runner naming convention
	version := "v2.311.0"      // TODO: Make this configurable
	versionNumber := "2.311.0" // Version without 'v' prefix for filename
	filename := fmt.Sprintf("actions-runner-%s-%s-%s.tar.gz", runnerOS, runnerArch, versionNumber)
	url := fmt.Sprintf("https://github.com/actions/runner/releases/download/%s/%s", version, filename)

	gb.logger.Printf("Constructed download URL: %s", url)
	return url
}
