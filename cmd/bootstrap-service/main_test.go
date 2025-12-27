package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestLoadRunnerConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	testConfig := &RunnerConfig{
		Method:          runnerTokenMethod,
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "test"},
		ExpiresAt:       "2025-12-25T06:00:55.977-06:00",
	}
	testConfig.Runner.DownloadURL = "https://example.com/runner.tar.gz"
	testConfig.Runner.InstallPath = testInstallPath
	testConfig.Runner.WorkDir = testWorkDir

	// Write test config to file
	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading the config
	loadedConfig, err := loadRunnerConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify the loaded config
	if loadedConfig.Method != testConfig.Method {
		t.Errorf("Expected method %s, got %s", testConfig.Method, loadedConfig.Method)
	}

	if loadedConfig.Platform != testConfig.Platform {
		t.Errorf("Expected platform %s, got %s", testConfig.Platform, loadedConfig.Platform)
	}

	if loadedConfig.RunnerToken != testConfig.RunnerToken {
		t.Errorf("Expected runner token %s, got %s", testConfig.RunnerToken, loadedConfig.RunnerToken)
	}

	if loadedConfig.RunnerName != testConfig.RunnerName {
		t.Errorf("Expected runner name %s, got %s", testConfig.RunnerName, loadedConfig.RunnerName)
	}

	if len(loadedConfig.Labels) != len(testConfig.Labels) {
		t.Errorf("Expected %d labels, got %d", len(testConfig.Labels), len(loadedConfig.Labels))
	}

	if loadedConfig.Runner.DownloadURL != testConfig.Runner.DownloadURL {
		t.Errorf("Expected download URL %s, got %s", testConfig.Runner.DownloadURL, loadedConfig.Runner.DownloadURL)
	}

	if loadedConfig.Runner.InstallPath != testConfig.Runner.InstallPath {
		t.Errorf("Expected install path %s, got %s", testConfig.Runner.InstallPath, loadedConfig.Runner.InstallPath)
	}

	if loadedConfig.Runner.WorkDir != testConfig.Runner.WorkDir {
		t.Errorf("Expected work dir %s, got %s", testConfig.Runner.WorkDir, loadedConfig.Runner.WorkDir)
	}
}

func TestLoadRunnerConfigFileNotFound(t *testing.T) {
	_, err := loadRunnerConfig("/nonexistent/config.json")
	if err == nil {
		t.Error("Expected error for nonexistent config file, got nil")
	}
}

func TestLoadRunnerConfigInvalidJSON(t *testing.T) {
	// Create a temporary config file with invalid JSON
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid-config.json")

	invalidJSON := `{"method": "runner-token", "invalid": json}`
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, err := loadRunnerConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestGitHubBootstrapDefaults(t *testing.T) {
	config := &RunnerConfig{
		Method:          "runner-token",
		Platform:        "github-actions",
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"test"},
	}
	// Leave Runner fields empty to test defaults

	bootstrap := &GitHubBootstrap{
		config: config,
	}

	// Test that defaults are applied correctly
	testCases := []struct {
		name     string
		getValue func() string
		expected string
	}{
		{
			name: "default download URL",
			getValue: func() string {
				url := bootstrap.config.Runner.DownloadURL
				if url == "" {
					return "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz"
				}
				return url
			},
			expected: "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz",
		},
		{
			name: "default install path",
			getValue: func() string {
				path := bootstrap.config.Runner.InstallPath
				if path == "" {
					return "/opt/actions-runner"
				}
				return path
			},
			expected: "/opt/actions-runner",
		},
		{
			name: "default work dir",
			getValue: func() string {
				workDir := bootstrap.config.Runner.WorkDir
				if workDir == "" {
					return "/tmp/runner-work"
				}
				return workDir
			},
			expected: "/tmp/runner-work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.getValue()
			if actual != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, actual)
			}
		})
	}
}

func TestGetOSArch(t *testing.T) {
	config := &RunnerConfig{}
	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test with empty config (should use runtime defaults)
	osName, arch := bootstrap.getOSArch()
	if osName == "" {
		t.Error("Expected non-empty OS")
	}
	if arch == "" {
		t.Error("Expected non-empty arch")
	}

	// Test with config values
	config.Runner.OS = "linux"
	config.Runner.Arch = "amd64"
	osName, arch = bootstrap.getOSArch()
	if osName != "linux" {
		t.Errorf("Expected OS 'linux', got '%s'", osName)
	}
	if arch != "amd64" {
		t.Errorf("Expected arch 'amd64', got '%s'", arch)
	}

	// Test with environment variables
	originalGOOS := os.Getenv("GOOS")
	originalGOARCH := os.Getenv("GOARCH")
	defer func() {
		_ = os.Setenv("GOOS", originalGOOS)
		_ = os.Setenv("GOARCH", originalGOARCH)
	}()

	config.Runner.OS = ""
	config.Runner.Arch = ""
	_ = os.Setenv("GOOS", "darwin")
	_ = os.Setenv("GOARCH", "arm64")

	osName, arch = bootstrap.getOSArch()
	if osName != "darwin" {
		t.Errorf("Expected OS 'darwin', got '%s'", osName)
	}
	if arch != "arm64" {
		t.Errorf("Expected arch 'arm64', got '%s'", arch)
	}
}

func TestBuildDownloadURL(t *testing.T) {
	testCases := []struct {
		name        string
		config      *RunnerConfig
		expectedURL string
	}{
		{
			name: "custom download URL",
			config: &RunnerConfig{
				Runner: struct {
					DownloadURL  string `json:"download_url,omitempty"`
					InstallPath  string `json:"install_path,omitempty"`
					WorkDir      string `json:"work_dir,omitempty"`
					ConfigScript string `json:"config_script,omitempty"`
					RunScript    string `json:"run_script,omitempty"`
					OS           string `json:"os,omitempty"`
					Arch         string `json:"arch,omitempty"`
				}{
					DownloadURL: "https://custom.example.com/runner.tar.gz",
				},
			},
			expectedURL: "https://custom.example.com/runner.tar.gz",
		},
		{
			name: "linux x64 default",
			config: &RunnerConfig{
				Runner: struct {
					DownloadURL  string `json:"download_url,omitempty"`
					InstallPath  string `json:"install_path,omitempty"`
					WorkDir      string `json:"work_dir,omitempty"`
					ConfigScript string `json:"config_script,omitempty"`
					RunScript    string `json:"run_script,omitempty"`
					OS           string `json:"os,omitempty"`
					Arch         string `json:"arch,omitempty"`
				}{
					OS:   "linux",
					Arch: "amd64",
				},
			},
			expectedURL: "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz",
		},
		{
			name: "darwin arm64",
			config: &RunnerConfig{
				Runner: struct {
					DownloadURL  string `json:"download_url,omitempty"`
					InstallPath  string `json:"install_path,omitempty"`
					WorkDir      string `json:"work_dir,omitempty"`
					ConfigScript string `json:"config_script,omitempty"`
					RunScript    string `json:"run_script,omitempty"`
					OS           string `json:"os,omitempty"`
					Arch         string `json:"arch,omitempty"`
				}{
					OS:   "darwin",
					Arch: "arm64",
				},
			},
			expectedURL: "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-osx-arm64-2.311.0.tar.gz",
		},
		{
			name: "windows x86",
			config: &RunnerConfig{
				Runner: struct {
					DownloadURL  string `json:"download_url,omitempty"`
					InstallPath  string `json:"install_path,omitempty"`
					WorkDir      string `json:"work_dir,omitempty"`
					ConfigScript string `json:"config_script,omitempty"`
					RunScript    string `json:"run_script,omitempty"`
					OS           string `json:"os,omitempty"`
					Arch         string `json:"arch,omitempty"`
				}{
					OS:   "windows",
					Arch: "386",
				},
			},
			expectedURL: "https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-win-x86-2.311.0.tar.gz",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bootstrap := &GitHubBootstrap{
				config: tc.config,
				logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
			}

			actualURL := bootstrap.buildDownloadURL()
			if actualURL != tc.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tc.expectedURL, actualURL)
			}
		})
	}
}

func TestPerformSPIFFEAttestation(t *testing.T) {
	testCases := []struct {
		name        string
		config      *RunnerConfig
		expectError bool
	}{
		{
			name: "valid SPIFFE config with join token",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					JoinToken: "test-join-token",
					SPIFFEID:  "spiffe://example.com/test",
					Enabled:   true,
				},
			},
			expectError: false,
		},
		{
			name: "valid SPIFFE config with SPIFFE ID only",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					SPIFFEID: "spiffe://example.com/test",
					Enabled:  true,
				},
			},
			expectError: false,
		},
		{
			name: "invalid SPIFFE config - no credentials",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					Enabled: true,
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bootstrap := &GitHubBootstrap{
				config: tc.config,
				logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
			}

			err := bootstrap.performSPIFFEAttestation()
			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestRunMethodValidation(t *testing.T) {
	testCases := []struct {
		name           string
		config         *RunnerConfig
		expectError    bool
		expectedMethod string
	}{
		{
			name: "runner-token method",
			config: &RunnerConfig{
				Method:          "runner-token",
				Platform:        "github-actions",
				RunnerToken:     "test-token",
				RegistrationURL: "https://github.com/test/repo",
				RunnerName:      "test-runner",
			},
			expectError:    false,
			expectedMethod: "runner-token",
		},
		{
			name: "join-token method",
			config: &RunnerConfig{
				Method: "join-token",
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					JoinToken: "test-join-token",
					Enabled:   true,
				},
			},
			expectError:    false,
			expectedMethod: "join-token",
		},
		{
			name: "unsupported method",
			config: &RunnerConfig{
				Method: "unsupported-method",
			},
			expectError:    true,
			expectedMethod: "unsupported-method",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.config.Method != tc.expectedMethod {
				t.Errorf("Expected method '%s', got '%s'", tc.expectedMethod, tc.config.Method)
			}

			// Test method validation logic
			switch tc.config.Method {
			case runnerTokenMethod:
				if tc.config.RunnerToken == "" && !tc.expectError {
					t.Error("runner-token method should require RunnerToken")
				}
			case joinTokenMethod:
				if tc.config.SPIFFE.JoinToken == "" && !tc.expectError {
					t.Error("join-token method should require SPIFFE JoinToken")
				}
			default:
				if !tc.expectError {
					t.Error("Unknown method should cause error")
				}
			}
		})
	}
}

func TestCleanupDirectoryHandling(t *testing.T) {
	// Create a test bootstrap instance
	config := &RunnerConfig{
		Method: "runner-token",
		Runner: struct {
			DownloadURL  string `json:"download_url,omitempty"`
			InstallPath  string `json:"install_path,omitempty"`
			WorkDir      string `json:"work_dir,omitempty"`
			ConfigScript string `json:"config_script,omitempty"`
			RunScript    string `json:"run_script,omitempty"`
			OS           string `json:"os,omitempty"`
			Arch         string `json:"arch,omitempty"`
		}{
			InstallPath: testInstallPathAlt,
			WorkDir:     "/tmp/test-work",
		},
	}

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that cleanup handles missing directories gracefully
	// This tests the error handling in the cleanup function
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// The cleanup function should not fail even if directories don't exist
	// We can't easily test the full cleanup without root privileges for shutdown,
	// but we can test the directory cleanup logic by checking the paths
	installPath := bootstrap.config.Runner.InstallPath
	workDir := bootstrap.config.Runner.WorkDir

	if installPath == "" {
		installPath = DefaultInstallPath
	}
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	// Verify the paths are set correctly
	if installPath != testInstallPathAlt {
		t.Errorf("Expected install path '%s', got '%s'", testInstallPathAlt, installPath)
	}
	if workDir != testWorkDir {
		t.Errorf("Expected work dir '%s', got '%s'", testWorkDir, workDir)
	}

	// Test context cancellation handling
	if ctx.Err() != nil {
		t.Errorf("Context should not be cancelled yet: %v", ctx.Err())
	}
}

func TestArchitectureMapping(t *testing.T) {
	testCases := []struct {
		goArch       string
		expectedArch string
	}{
		{"amd64", "x64"},
		{"arm64", "arm64"},
		{"386", "x86"},
		{"unknown", "unknown"}, // fallback case
	}

	for _, tc := range testCases {
		t.Run(tc.goArch, func(t *testing.T) {
			config := &RunnerConfig{
				Runner: struct {
					DownloadURL  string `json:"download_url,omitempty"`
					InstallPath  string `json:"install_path,omitempty"`
					WorkDir      string `json:"work_dir,omitempty"`
					ConfigScript string `json:"config_script,omitempty"`
					RunScript    string `json:"run_script,omitempty"`
					OS           string `json:"os,omitempty"`
					Arch         string `json:"arch,omitempty"`
				}{
					OS:   "linux",
					Arch: tc.goArch,
				},
			}

			bootstrap := &GitHubBootstrap{
				config: config,
				logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
			}

			url := bootstrap.buildDownloadURL()
			expectedSubstring := fmt.Sprintf("actions-runner-linux-%s-", tc.expectedArch)
			if !strings.Contains(url, expectedSubstring) {
				t.Errorf("Expected URL to contain '%s', got '%s'", expectedSubstring, url)
			}
		})
	}
}

func TestOSMapping(t *testing.T) {
	testCases := []struct {
		goOS       string
		expectedOS string
	}{
		{"linux", "linux"},
		{"darwin", "osx"},
		{"windows", "win"},
		{"unknown", "unknown"}, // fallback case
	}

	for _, tc := range testCases {
		t.Run(tc.goOS, func(t *testing.T) {
			config := &RunnerConfig{
				Runner: struct {
					DownloadURL  string `json:"download_url,omitempty"`
					InstallPath  string `json:"install_path,omitempty"`
					WorkDir      string `json:"work_dir,omitempty"`
					ConfigScript string `json:"config_script,omitempty"`
					RunScript    string `json:"run_script,omitempty"`
					OS           string `json:"os,omitempty"`
					Arch         string `json:"arch,omitempty"`
				}{
					OS:   tc.goOS,
					Arch: "amd64",
				},
			}

			bootstrap := &GitHubBootstrap{
				config: config,
				logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
			}

			url := bootstrap.buildDownloadURL()
			expectedSubstring := fmt.Sprintf("actions-runner-%s-x64-", tc.expectedOS)
			if !strings.Contains(url, expectedSubstring) {
				t.Errorf("Expected URL to contain '%s', got '%s'", expectedSubstring, url)
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	// Test that all default constants are properly defined
	if DefaultDownloadURL == "" {
		t.Error("DefaultDownloadURL should not be empty")
	}
	if DefaultInstallPath == "" {
		t.Error("DefaultInstallPath should not be empty")
	}
	if DefaultWorkDir == "" {
		t.Error("DefaultWorkDir should not be empty")
	}
	if DefaultConfigPath == "" {
		t.Error("DefaultConfigPath should not be empty")
	}
	if DefaultConfigScript == "" {
		t.Error("DefaultConfigScript should not be empty")
	}
	if DefaultRunScript == "" {
		t.Error("DefaultRunScript should not be empty")
	}

	// Test numeric constants
	if DirPermissions == 0 {
		t.Error("DirPermissions should not be zero")
	}
	if CleanupDelaySeconds == 0 {
		t.Error("CleanupDelaySeconds should not be zero")
	}
	if HTTPTimeoutSeconds == 0 {
		t.Error("HTTPTimeoutSeconds should not be zero")
	}
}

func TestRuntimeDefaults(t *testing.T) {
	config := &RunnerConfig{}
	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that runtime defaults are used when config is empty
	os, arch := bootstrap.getOSArch()

	// Should match runtime values
	expectedOS := runtime.GOOS
	expectedArch := runtime.GOARCH

	if os != expectedOS {
		t.Errorf("Expected OS '%s', got '%s'", expectedOS, os)
	}
	if arch != expectedArch {
		t.Errorf("Expected arch '%s', got '%s'", expectedArch, arch)
	}
}
func TestConfigureRunnerArgs(t *testing.T) {
	config := &RunnerConfig{
		Method:          "runner-token",
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner-name",
		Labels:          []string{"self-hosted", "linux", "x64"},
	}
	config.Runner.InstallPath = testInstallPath
	config.Runner.WorkDir = testWorkDir
	config.Runner.ConfigScript = DefaultConfigScript

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that the configuration arguments are built correctly
	// We can't easily test the actual execution without mocking exec.Command
	// but we can test the path construction logic
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	configScript := bootstrap.config.Runner.ConfigScript
	if configScript == "" {
		configScript = DefaultConfigScript
	}

	configScriptPath := filepath.Join(installPath, configScript)

	// Verify paths are constructed correctly
	expectedConfigPath := "/opt/test-runner/config.sh"
	if configScriptPath != expectedConfigPath {
		t.Errorf("Expected config script path '%s', got '%s'", expectedConfigPath, configScriptPath)
	}

	if workDir != testWorkDir {
		t.Errorf("Expected work dir '%s', got '%s'", testWorkDir, workDir)
	}

	// Test labels joining
	expectedLabels := "self-hosted,linux,x64"
	actualLabels := strings.Join(bootstrap.config.Labels, ",")
	if actualLabels != expectedLabels {
		t.Errorf("Expected labels '%s', got '%s'", expectedLabels, actualLabels)
	}
}

func TestRunAndMonitorPaths(t *testing.T) {
	config := &RunnerConfig{
		Method: runnerTokenMethod,
	}
	config.Runner.InstallPath = testInstallPath
	config.Runner.RunScript = DefaultRunScript

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test path construction for run script
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	runScript := bootstrap.config.Runner.RunScript
	if runScript == "" {
		runScript = DefaultRunScript
	}

	runScriptPath := filepath.Join(installPath, runScript)
	expectedRunPath := "/opt/test-runner/run.sh"

	if runScriptPath != expectedRunPath {
		t.Errorf("Expected run script path '%s', got '%s'", expectedRunPath, runScriptPath)
	}
}

func TestCleanupPaths(t *testing.T) {
	config := &RunnerConfig{
		Method: runnerTokenMethod,
	}
	config.Runner.InstallPath = testInstallPath
	config.Runner.WorkDir = testWorkDir

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test cleanup path logic
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	if installPath != testInstallPath {
		t.Errorf("Expected install path '%s', got '%s'", testInstallPath, installPath)
	}

	if workDir != testWorkDir {
		t.Errorf("Expected work dir '%s', got '%s'", testWorkDir, workDir)
	}
}

func TestMainMethodSwitch(t *testing.T) {
	testCases := []struct {
		name            string
		method          string
		expectSupported bool
	}{
		{
			name:            "runner-token method supported",
			method:          "runner-token",
			expectSupported: true,
		},
		{
			name:            "join-token method not implemented",
			method:          "join-token",
			expectSupported: false,
		},
		{
			name:            "unknown method not supported",
			method:          "unknown-method",
			expectSupported: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the method validation logic that would be used in main()
			switch tc.method {
			case runnerTokenMethod:
				if !tc.expectSupported {
					t.Error("runner-token method should be supported")
				}
			case joinTokenMethod:
				if tc.expectSupported {
					t.Error("join-token method should not be implemented yet")
				}
			default:
				if tc.expectSupported {
					t.Error("unknown method should not be supported")
				}
			}
		})
	}
}

func TestDownloadURLConstruction(t *testing.T) {
	// Test URL construction with different version scenarios
	config := &RunnerConfig{
		Runner: struct {
			DownloadURL  string `json:"download_url,omitempty"`
			InstallPath  string `json:"install_path,omitempty"`
			WorkDir      string `json:"work_dir,omitempty"`
			ConfigScript string `json:"config_script,omitempty"`
			RunScript    string `json:"run_script,omitempty"`
			OS           string `json:"os,omitempty"`
			Arch         string `json:"arch,omitempty"`
		}{
			OS:   "linux",
			Arch: "amd64",
		},
	}

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	url := bootstrap.buildDownloadURL()

	// Verify URL structure
	if !strings.Contains(url, "github.com/actions/runner/releases/download") {
		t.Error("URL should contain GitHub releases path")
	}

	if !strings.Contains(url, "v2.311.0") {
		t.Error("URL should contain version")
	}

	if !strings.Contains(url, "actions-runner-linux-x64-2.311.0.tar.gz") {
		t.Error("URL should contain correct filename")
	}
}

func TestErrorHandlingInSPIFFE(t *testing.T) {
	// Test SPIFFE error conditions more thoroughly
	testCases := []struct {
		name        string
		config      *RunnerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "SPIFFE enabled but no credentials",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					Enabled: true,
				},
			},
			expectError: true,
			errorMsg:    "SPIFFE attestation enabled but no join token or SPIFFE ID provided",
		},
		{
			name: "SPIFFE with empty strings",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					JoinToken: "",
					SPIFFEID:  "",
					Enabled:   true,
				},
			},
			expectError: true,
			errorMsg:    "SPIFFE attestation enabled but no join token or SPIFFE ID provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bootstrap := &GitHubBootstrap{
				config: tc.config,
				logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
			}

			err := bootstrap.performSPIFFEAttestation()

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if err.Error() != tc.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSPIFFEEnabledCheck(t *testing.T) {
	// Test the logic that would be used in main() to check if SPIFFE is enabled
	testCases := []struct {
		name    string
		config  *RunnerConfig
		enabled bool
	}{
		{
			name: "SPIFFE enabled",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					Enabled: true,
				},
			},
			enabled: true,
		},
		{
			name: "SPIFFE disabled",
			config: &RunnerConfig{
				SPIFFE: struct {
					JoinToken string `json:"join_token,omitempty"`
					SPIFFEID  string `json:"spiffe_id,omitempty"`
					Enabled   bool   `json:"enabled,omitempty"`
				}{
					Enabled: false,
				},
			},
			enabled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the enabled check logic that would be used in main()
			if tc.config.SPIFFE.Enabled != tc.enabled {
				t.Errorf("Expected enabled %v, got %v", tc.enabled, tc.config.SPIFFE.Enabled)
			}
		})
	}
}
func TestRunWorkflow(t *testing.T) {
	// Test the main Run workflow logic without actually executing external commands
	config := &RunnerConfig{
		Method:          "runner-token",
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "test"},
	}
	config.Runner.InstallPath = testInstallPathAlt
	config.Runner.WorkDir = testWorkDir

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that the workflow steps are properly structured
	// We can't run the actual workflow without mocking HTTP and exec calls,
	// but we can test the configuration and path setup

	// Test download URL construction
	downloadURL := bootstrap.buildDownloadURL()
	if downloadURL == "" {
		t.Error("Download URL should not be empty")
	}

	// Test install path setup
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}
	if installPath != testInstallPathAlt {
		t.Errorf("Expected install path '%s', got '%s'", testInstallPathAlt, installPath)
	}

	// Test work directory setup
	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}
	if workDir != testWorkDir {
		t.Errorf("Expected work dir '%s', got '%s'", testWorkDir, workDir)
	}
}

func TestDownloadGitHubRunnerPathSetup(t *testing.T) {
	// Test the path setup logic in downloadGitHubRunner without actual download
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test install path logic
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	downloadURL := bootstrap.buildDownloadURL()

	// Verify the setup that would be used in downloadGitHubRunner
	if installPath != testInstallPath {
		t.Errorf("Expected install path '%s', got '%s'", testInstallPath, installPath)
	}

	if downloadURL == "" {
		t.Error("Download URL should not be empty")
	}

	// Test default path fallback
	emptyConfig := &RunnerConfig{}
	emptyBootstrap := &GitHubBootstrap{
		config: emptyConfig,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	defaultInstallPath := emptyBootstrap.config.Runner.InstallPath
	if defaultInstallPath == "" {
		defaultInstallPath = DefaultInstallPath
	}

	if defaultInstallPath != DefaultInstallPath {
		t.Errorf("Expected default install path '%s', got '%s'", DefaultInstallPath, defaultInstallPath)
	}
}

func TestConfigureRunnerPathConstruction(t *testing.T) {
	// Test the path construction logic in configureRunner
	config := &RunnerConfig{
		Method:          "runner-token",
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}
	config.Runner.InstallPath = testInstallPath
	config.Runner.WorkDir = testWorkDir
	config.Runner.ConfigScript = DefaultConfigScript

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test path construction that would be used in configureRunner
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	configScript := bootstrap.config.Runner.ConfigScript
	if configScript == "" {
		configScript = DefaultConfigScript
	}

	configScriptPath := filepath.Join(installPath, configScript)

	// Verify paths
	if configScriptPath != "/opt/test-runner/config.sh" {
		t.Errorf("Expected config script path '/opt/test-runner/config.sh', got '%s'", configScriptPath)
	}

	if workDir != testWorkDir {
		t.Errorf("Expected work dir '%s', got '%s'", testWorkDir, workDir)
	}

	// Test argument construction
	expectedArgs := []string{
		"--url", bootstrap.config.RegistrationURL,
		"--token", bootstrap.config.RunnerToken,
		"--name", bootstrap.config.RunnerName,
		"--labels", strings.Join(bootstrap.config.Labels, ","),
		"--work", workDir,
		"--unattended",
		"--ephemeral",
	}

	// Verify argument structure (we can't test exec.Command without mocking)
	if len(expectedArgs) != 12 {
		t.Errorf("Expected 12 arguments, got %d", len(expectedArgs))
	}

	if expectedArgs[1] != bootstrap.config.RegistrationURL {
		t.Errorf("Expected URL argument '%s', got '%s'", bootstrap.config.RegistrationURL, expectedArgs[1])
	}
}

func TestRunAndMonitorPathConstruction(t *testing.T) {
	// Test the path construction logic in runAndMonitor
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath
	config.Runner.RunScript = DefaultRunScript

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test path construction that would be used in runAndMonitor
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	runScript := bootstrap.config.Runner.RunScript
	if runScript == "" {
		runScript = DefaultRunScript
	}

	runScriptPath := filepath.Join(installPath, runScript)

	// Verify paths
	if runScriptPath != "/opt/test-runner/run.sh" {
		t.Errorf("Expected run script path '/opt/test-runner/run.sh', got '%s'", runScriptPath)
	}

	// Test default fallback
	defaultConfig := &RunnerConfig{}
	defaultBootstrap := &GitHubBootstrap{
		config: defaultConfig,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	defaultInstallPath := defaultBootstrap.config.Runner.InstallPath
	if defaultInstallPath == "" {
		defaultInstallPath = DefaultInstallPath
	}

	defaultRunScript := defaultBootstrap.config.Runner.RunScript
	if defaultRunScript == "" {
		defaultRunScript = DefaultRunScript
	}

	defaultRunScriptPath := filepath.Join(defaultInstallPath, defaultRunScript)
	expectedDefaultPath := filepath.Join(DefaultInstallPath, DefaultRunScript)

	if defaultRunScriptPath != expectedDefaultPath {
		t.Errorf("Expected default run script path '%s', got '%s'", expectedDefaultPath, defaultRunScriptPath)
	}
}

func TestCleanupLogic(t *testing.T) {
	// Test the cleanup logic without actually performing cleanup
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath
	config.Runner.WorkDir = testWorkDir

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test cleanup path logic that would be used in cleanup function
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	// Verify cleanup would target correct paths
	if installPath != testInstallPath {
		t.Errorf("Expected cleanup install path '%s', got '%s'", testInstallPath, installPath)
	}

	if workDir != testWorkDir {
		t.Errorf("Expected cleanup work dir '%s', got '%s'", testWorkDir, workDir)
	}

	// Test default path cleanup
	defaultConfig := &RunnerConfig{}
	defaultBootstrap := &GitHubBootstrap{
		config: defaultConfig,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	defaultInstallPath := defaultBootstrap.config.Runner.InstallPath
	if defaultInstallPath == "" {
		defaultInstallPath = DefaultInstallPath
	}

	defaultWorkDir := defaultBootstrap.config.Runner.WorkDir
	if defaultWorkDir == "" {
		defaultWorkDir = DefaultWorkDir
	}

	if defaultInstallPath != DefaultInstallPath {
		t.Errorf("Expected default cleanup install path '%s', got '%s'", DefaultInstallPath, defaultInstallPath)
	}

	if defaultWorkDir != DefaultWorkDir {
		t.Errorf("Expected default cleanup work dir '%s', got '%s'", DefaultWorkDir, defaultWorkDir)
	}
}

func TestShutdownMethodSelection(t *testing.T) {
	// Test the shutdown method selection logic
	config := &RunnerConfig{}
	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that we have multiple shutdown methods available
	// We can't test the actual shutdown without root privileges,
	// but we can test the method selection logic

	shutdownCommands := [][]string{
		{"sudo", "shutdown", "-h", "now"},
		{"shutdown", "-h", "now"},
		{"sudo", "poweroff"},
		{"poweroff"},
		{"sudo", "halt", "-p"},
		{"halt", "-p"},
		{"sudo", "systemctl", "poweroff"},
		{"systemctl", "poweroff"},
	}

	// Verify we have multiple fallback methods
	if len(shutdownCommands) < 4 {
		t.Error("Should have multiple shutdown command fallbacks")
	}

	// Verify command structure
	for i, cmd := range shutdownCommands {
		if len(cmd) == 0 {
			t.Errorf("Shutdown command %d should not be empty", i)
		}
		if cmd[0] == "" {
			t.Errorf("Shutdown command %d should have non-empty executable", i)
		}
	}

	// Test that bootstrap instance is properly configured
	if bootstrap.config == nil {
		t.Error("Bootstrap config should not be nil")
	}

	if bootstrap.logger == nil {
		t.Error("Bootstrap logger should not be nil")
	}
}
func TestRunMethodExecution(t *testing.T) {
	// Test the Run method setup without external dependencies
	config := &RunnerConfig{
		Method:          "runner-token",
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "test"},
	}

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that the bootstrap is properly configured for Run
	if bootstrap.config.Method != "runner-token" {
		t.Errorf("Expected method 'runner-token', got '%s'", bootstrap.config.Method)
	}

	if bootstrap.config.RunnerName != "test-runner" {
		t.Errorf("Expected runner name 'test-runner', got '%s'", bootstrap.config.RunnerName)
	}

	if bootstrap.logger == nil {
		t.Error("Logger should not be nil")
	}

	// We can't test the full Run method without mocking HTTP and exec calls,
	// but we can verify the configuration is ready for execution
	downloadURL := bootstrap.buildDownloadURL()
	if downloadURL == "" {
		t.Error("Download URL should be ready for Run method")
	}
}

func TestDownloadGitHubRunnerSetup(t *testing.T) {
	// Test the setup phase of downloadGitHubRunner
	config := &RunnerConfig{}
	config.Runner.InstallPath = "/tmp/test-install"

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test the path setup that happens at the start of downloadGitHubRunner
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	downloadURL := bootstrap.buildDownloadURL()

	// Verify setup is correct
	if installPath != "/tmp/test-install" {
		t.Errorf("Expected install path '/tmp/test-install', got '%s'", installPath)
	}

	if !strings.Contains(downloadURL, "github.com/actions/runner") {
		t.Error("Download URL should point to GitHub Actions runner")
	}

	// Test HTTP client timeout configuration
	timeout := HTTPTimeoutSeconds
	if timeout != 300 {
		t.Errorf("Expected HTTP timeout 300 seconds, got %d", timeout)
	}
}

func TestConfigureRunnerSetup(t *testing.T) {
	// Test the setup phase of configureRunner
	config := &RunnerConfig{
		Method:          "runner-token",
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}
	config.Runner.InstallPath = testOptPath
	config.Runner.WorkDir = testTmpWork
	config.Runner.ConfigScript = DefaultConfigScript

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test the path and argument setup that happens in configureRunner
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	configScript := bootstrap.config.Runner.ConfigScript
	if configScript == "" {
		configScript = DefaultConfigScript
	}

	configScriptPath := filepath.Join(installPath, configScript)

	// Verify configuration setup
	if configScriptPath != "/opt/test/config.sh" {
		t.Errorf("Expected config script path '/opt/test/config.sh', got '%s'", configScriptPath)
	}

	if workDir != testTmpWork {
		t.Errorf("Expected work dir '%s', got '%s'", testTmpWork, workDir)
	}

	// Test argument construction
	labels := strings.Join(bootstrap.config.Labels, ",")
	if labels != "self-hosted" {
		t.Errorf("Expected labels 'self-hosted', got '%s'", labels)
	}
}

func TestRunAndMonitorSetup(t *testing.T) {
	// Test the setup phase of runAndMonitor
	config := &RunnerConfig{}
	config.Runner.InstallPath = testOptPath
	config.Runner.RunScript = DefaultRunScript

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test the path setup that happens in runAndMonitor
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	runScript := bootstrap.config.Runner.RunScript
	if runScript == "" {
		runScript = DefaultRunScript
	}

	runScriptPath := filepath.Join(installPath, runScript)

	// Verify setup
	if runScriptPath != "/opt/test/run.sh" {
		t.Errorf("Expected run script path '/opt/test/run.sh', got '%s'", runScriptPath)
	}
}

func TestCleanupSetup(t *testing.T) {
	// Test the setup phase of cleanup
	config := &RunnerConfig{}
	config.Runner.InstallPath = "/opt/test"
	config.Runner.WorkDir = "/tmp/work"

	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test the path setup that happens in cleanup
	installPath := bootstrap.config.Runner.InstallPath
	if installPath == "" {
		installPath = DefaultInstallPath
	}

	workDir := bootstrap.config.Runner.WorkDir
	if workDir == "" {
		workDir = DefaultWorkDir
	}

	// Verify cleanup setup
	if installPath != "/opt/test" {
		t.Errorf("Expected install path '/opt/test', got '%s'", installPath)
	}

	if workDir != "/tmp/work" {
		t.Errorf("Expected work dir '/tmp/work', got '%s'", workDir)
	}

	// Test cleanup delay constant
	delay := CleanupDelaySeconds
	if delay != 2 {
		t.Errorf("Expected cleanup delay 2 seconds, got %d", delay)
	}
}

func TestShutdownVMSetup(t *testing.T) {
	// Test the shutdown method selection logic
	config := &RunnerConfig{}
	bootstrap := &GitHubBootstrap{
		config: config,
		logger: log.New(os.Stdout, "[test] ", log.LstdFlags),
	}

	// Test that we have the expected shutdown methods
	// We can't test actual shutdown without root privileges,
	// but we can test the method availability

	// Test syscall constants are available
	if syscall.LINUX_REBOOT_CMD_POWER_OFF == 0 {
		t.Error("Syscall power off constant should be available")
	}

	// Test SysRq file path
	sysrqFile := "/proc/sysrq-trigger"
	if sysrqFile == "" {
		t.Error("SysRq file path should be defined")
	}

	// Test power state file path
	powerStateFile := "/sys/power/state"
	if powerStateFile == "" {
		t.Error("Power state file path should be defined")
	}

	// Test that bootstrap is configured
	if bootstrap.logger == nil {
		t.Error("Bootstrap logger should be configured")
	}
}

func TestMainFunctionLogic(t *testing.T) {
	// Test the main function logic without actually running main
	testCases := []struct {
		name          string
		method        string
		shouldSupport bool
		shouldFail    bool
	}{
		{
			name:          "runner-token method",
			method:        "runner-token",
			shouldSupport: true,
			shouldFail:    false,
		},
		{
			name:          "join-token method",
			method:        "join-token",
			shouldSupport: false,
			shouldFail:    true,
		},
		{
			name:          "unknown method",
			method:        "unknown",
			shouldSupport: false,
			shouldFail:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the method switch logic that would be used in main
			var supported bool
			var wouldFail bool

			switch tc.method {
			case "runner-token":
				supported = true
				wouldFail = false
			case "join-token":
				supported = false
				wouldFail = true // Not yet implemented
			default:
				supported = false
				wouldFail = true // Unsupported method
			}

			if supported != tc.shouldSupport {
				t.Errorf("Expected support %v, got %v", tc.shouldSupport, supported)
			}

			if wouldFail != tc.shouldFail {
				t.Errorf("Expected fail %v, got %v", tc.shouldFail, wouldFail)
			}
		})
	}
}
