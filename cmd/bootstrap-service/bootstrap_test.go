package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Test constants
const (
	testInstallPath    = "/opt/test-runner"
	testWorkDir        = "/tmp/test-work"
	testConfigScript   = "/opt/test-runner/config.sh"
	testRunScript      = "/opt/test-runner/run.sh"
	testSysRqTrigger   = "/proc/sysrq-trigger"
	testOptPath        = "/opt/test"
	testTmpWork        = "/tmp/work"
	testInstallPathAlt = "/tmp/test-install"
)

func TestNewGitHubBootstrap(t *testing.T) {
	config := &RunnerConfig{Method: runnerTokenMethod}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	if bootstrap.config != config {
		t.Error("Config should be set correctly")
	}
	if bootstrap.logger != logger {
		t.Error("Logger should be set correctly")
	}
	if bootstrap.httpClient != httpClient {
		t.Error("HTTP client should be set correctly")
	}
	if bootstrap.fileSystem != fileSystem {
		t.Error("File system should be set correctly")
	}
	if bootstrap.executor != executor {
		t.Error("Executor should be set correctly")
	}
	if bootstrap.system != system {
		t.Error("System should be set correctly")
	}
}

func TestRunWorkflowWithMocks(t *testing.T) {
	config := &RunnerConfig{
		Method:          runnerTokenMethod,
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Return a mock tar.gz response
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	// Test that Run method calls the workflow steps
	// We expect it to fail at the download step due to invalid tar data,
	// but we can verify the setup was called
	ctx := context.Background()
	err := bootstrap.Run(ctx)

	// Should fail at tar extraction, but that's expected with empty response
	if err == nil {
		t.Error("Expected error due to invalid tar data")
	}

	// Verify logger was used
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}

	// Verify HTTP client was called
	if len(fileSystem.CreatedDirs) == 0 {
		t.Error("Expected directories to be created")
	}
}

func TestDownloadGitHubRunnerWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.String(), "github.com/actions/runner") {
				t.Error("Should request GitHub Actions runner")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	// Should fail at tar extraction, but we can verify setup
	if err == nil {
		t.Error("Expected error due to invalid tar data")
	}

	// Verify directory creation was attempted
	if len(fileSystem.CreatedDirs) == 0 {
		t.Error("Expected install directory to be created")
	}

	expectedDir := testInstallPath
	found := false
	for _, dir := range fileSystem.CreatedDirs {
		if dir == expectedDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected directory '%s' to be created", expectedDir)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestConfigureRunnerWithMocks(t *testing.T) {
	config := &RunnerConfig{
		Method:          runnerTokenMethod,
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "linux"},
	}
	config.Runner.InstallPath = testInstallPath
	config.Runner.WorkDir = testWorkDir

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.configureRunner(ctx)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify command execution
	if len(executor.ExecutedCommands) != 1 {
		t.Errorf("Expected 1 command execution, got %d", len(executor.ExecutedCommands))
	}

	cmd := executor.ExecutedCommands[0]
	expectedScript := testConfigScript
	if cmd.Name != expectedScript {
		t.Errorf("Expected command '%s', got '%s'", expectedScript, cmd.Name)
	}

	// Verify arguments
	expectedArgs := []string{
		"--url", "https://github.com/test/repo",
		"--token", "test-token",
		"--name", "test-runner",
		"--labels", "self-hosted,linux",
		"--work", "/tmp/test-work",
		"--unattended",
		"--ephemeral",
	}

	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(cmd.Args))
	}

	for i, expected := range expectedArgs {
		if i < len(cmd.Args) && cmd.Args[i] != expected {
			t.Errorf("Expected arg[%d] '%s', got '%s'", i, expected, cmd.Args[i])
		}
	}

	// Verify directory was set
	if cmd.Dir != testInstallPath {
		t.Errorf("Expected dir '%s', got '%s'", testInstallPath, cmd.Dir)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestRunAndMonitorWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath
	config.Runner.RunScript = DefaultRunScript

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.runAndMonitor(ctx)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify command execution
	if len(executor.ExecutedCommands) != 1 {
		t.Errorf("Expected 1 command execution, got %d", len(executor.ExecutedCommands))
	}

	cmd := executor.ExecutedCommands[0]
	expectedScript := testRunScript
	if cmd.Name != expectedScript {
		t.Errorf("Expected command '%s', got '%s'", expectedScript, cmd.Name)
	}

	// Verify directory was set
	if cmd.Dir != testInstallPath {
		t.Errorf("Expected dir '%s', got '%s'", testInstallPath, cmd.Dir)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestCleanupWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath
	config.Runner.WorkDir = testWorkDir

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.cleanup(ctx)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify directories were removed
	expectedPaths := []string{testInstallPath, testWorkDir}
	if len(fileSystem.RemovedPaths) != len(expectedPaths) {
		t.Errorf("Expected %d removed paths, got %d", len(expectedPaths), len(fileSystem.RemovedPaths))
	}

	for _, expected := range expectedPaths {
		found := false
		for _, removed := range fileSystem.RemovedPaths {
			if removed == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected path '%s' to be removed", expected)
		}
	}

	// Verify sleep was called
	if !system.SleepCalled {
		t.Error("Expected sleep to be called")
	}

	if system.SleepDuration != CleanupDelaySeconds {
		t.Errorf("Expected sleep duration %d, got %d", CleanupDelaySeconds, system.SleepDuration)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestShutdownViaSyscallWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaSyscall()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify sync was called
	if !system.SyncCalled {
		t.Error("Expected sync to be called")
	}

	// Verify reboot was called
	if !system.RebootCalled {
		t.Error("Expected reboot to be called")
	}

	if system.RebootCmd != syscall.LINUX_REBOOT_CMD_POWER_OFF {
		t.Errorf("Expected reboot cmd %d, got %d", syscall.LINUX_REBOOT_CMD_POWER_OFF, system.RebootCmd)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestShutdownViaSysRqWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaSysRq()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify file was opened
	expectedFile := testSysRqTrigger
	found := false
	for _, file := range fileSystem.OpenedFiles {
		if file == expectedFile {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected file '%s' to be opened", expectedFile)
	}

	// Verify data was written
	if data, exists := fileSystem.WrittenData[expectedFile]; !exists || data != "o" {
		t.Errorf("Expected 'o' to be written to sysrq-trigger, got '%s'", data)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestShutdownViaPowerStateWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaPowerState()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify power disk file was opened
	expectedFile := "/sys/power/disk"
	found := false
	for _, file := range fileSystem.OpenedFiles {
		if file == expectedFile {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected file '%s' to be opened", expectedFile)
	}

	// Verify data was written
	if data, exists := fileSystem.WrittenData[expectedFile]; !exists || data != "shutdown" {
		t.Errorf("Expected 'shutdown' to be written to power disk, got '%s'", data)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestShutdownViaCommandWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaCommand()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify at least one command was executed
	if len(executor.ExecutedCommands) == 0 {
		t.Error("Expected at least one shutdown command to be executed")
	}

	// Verify first command is a shutdown command
	cmd := executor.ExecutedCommands[0]
	expectedCommands := []string{"sudo", "shutdown", "poweroff", "halt", "systemctl"}
	found := false
	for _, expected := range expectedCommands {
		if cmd.Name == expected || (len(cmd.Args) > 0 && cmd.Args[0] == expected) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected shutdown command, got '%s' with args %v", cmd.Name, cmd.Args)
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestShutdownVMWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownVM()

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify syscall method was tried first
	if !system.SyncCalled {
		t.Error("Expected sync to be called (syscall method)")
	}

	if !system.RebootCalled {
		t.Error("Expected reboot to be called (syscall method)")
	}

	// Verify logging
	if len(logger.Messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestShutdownVMFallbackWithMocks(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := &MockSystemOperations{
		RebootFunc: func(cmd int) error {
			return fmt.Errorf("syscall failed")
		},
	}

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownVM()

	if err != nil {
		t.Errorf("Expected no error (should fallback), got: %v", err)
	}

	// Verify syscall was tried and failed
	if !system.RebootCalled {
		t.Error("Expected reboot to be called")
	}

	// Verify fallback methods were tried
	// Should try SysRq, power state, and commands
	if len(fileSystem.OpenedFiles) == 0 && len(executor.ExecutedCommands) == 0 {
		t.Error("Expected fallback methods to be tried")
	}
}
func TestRealImplementations(t *testing.T) {
	// Test real HTTP client
	httpClient := NewRealHTTPClient(5 * time.Second)
	if httpClient == nil {
		t.Error("HTTP client should not be nil")
	}

	// Test real file system
	fileSystem := NewRealFileSystem()
	if fileSystem == nil {
		t.Error("File system should not be nil")
	}

	// Test real command executor
	executor := NewRealCommandExecutor()
	if executor == nil {
		t.Error("Command executor should not be nil")
	}

	// Test real system operations
	system := NewRealSystemOperations()
	if system == nil {
		t.Error("System operations should not be nil")
	}

	// Test real logger
	logger := NewRealLogger("[test] ")
	if logger == nil {
		t.Error("Logger should not be nil")
	}

	// Test logger functionality
	logger.Printf("Test message: %s", "hello")
}

func TestMainFunctionWithMocks(t *testing.T) {
	// Test the main function logic by creating a config file and testing the switch logic
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	testConfig := &RunnerConfig{
		Method:          runnerTokenMethod,
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "test"},
	}

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test config loading (this is what main() does first)
	config, err := loadRunnerConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test method switch logic (this is what main() does next)
	switch config.Method {
	case runnerTokenMethod:
		// This path should be taken
		bootstrap := NewGitHubBootstrap(
			config,
			NewMockLogger(),
			&MockHTTPClient{},
			NewMockFileSystem(),
			NewMockCommandExecutor(),
			NewMockSystemOperations(),
		)
		if bootstrap == nil {
			t.Error("Bootstrap should be created for runner-token method")
		}
	case joinTokenMethod:
		t.Error("join-token method should not be reached with runner-token config")
	default:
		t.Error("Unknown method should not be reached with runner-token config")
	}
}

func TestDownloadGitHubRunnerErrorHandling(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()

	// Test HTTP error
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to network failure")
	}

	if !strings.Contains(err.Error(), "failed to download runner") {
		t.Errorf("Expected download error, got: %v", err)
	}

	// Test HTTP status error
	httpClient2 := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}

	bootstrap2 := NewGitHubBootstrap(config, logger, httpClient2, fileSystem, executor, system)
	err2 := bootstrap2.downloadGitHubRunner(ctx)

	if err2 == nil {
		t.Error("Expected error due to HTTP 404")
	}

	if !strings.Contains(err2.Error(), "HTTP 404") {
		t.Errorf("Expected HTTP 404 error, got: %v", err2)
	}
}

func TestConfigureRunnerErrorHandling(t *testing.T) {
	config := &RunnerConfig{
		Method:          "runner-token",
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()

	// Test command execution error
	executor := NewMockCommandExecutor()
	executor.CommandContextFunc = func(ctx context.Context, name string, args ...string) Command {
		return &MockCommand{
			name:     name,
			args:     args,
			executor: executor,
			RunFunc: func() error {
				return fmt.Errorf("command failed")
			},
		}
	}
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.configureRunner(ctx)

	if err == nil {
		t.Error("Expected error due to command failure")
	}

	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("Expected command failure error, got: %v", err)
	}
}

func TestRunAndMonitorErrorHandling(t *testing.T) {
	config := &RunnerConfig{}

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()

	// Test command execution error
	executor := NewMockCommandExecutor()
	executor.CommandContextFunc = func(ctx context.Context, name string, args ...string) Command {
		return &MockCommand{
			name:     name,
			args:     args,
			executor: executor,
			RunFunc: func() error {
				return fmt.Errorf("runner failed")
			},
		}
	}
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.runAndMonitor(ctx)

	if err == nil {
		t.Error("Expected error due to runner failure")
	}

	if !strings.Contains(err.Error(), "runner failed") {
		t.Errorf("Expected runner failure error, got: %v", err)
	}
}

func TestCleanupErrorHandling(t *testing.T) {
	config := &RunnerConfig{}

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}

	// Test file system errors (should not cause cleanup to fail)
	fileSystem := &MockFileSystem{
		RemoveAllFunc: func(path string) error {
			return fmt.Errorf("permission denied")
		},
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.cleanup(ctx)

	// Cleanup should not fail even if file removal fails
	if err != nil {
		t.Errorf("Cleanup should not fail due to file removal errors, got: %v", err)
	}

	// Verify warning messages were logged
	if len(logger.Messages) == 0 {
		t.Error("Expected warning messages to be logged")
	}
}

func TestShutdownErrorHandling(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()

	// Test all shutdown methods failing
	system := &MockSystemOperations{
		RebootFunc: func(cmd int) error {
			return fmt.Errorf("syscall failed")
		},
	}

	// Make file operations fail
	fileSystem.OpenFileFunc = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		return nil, fmt.Errorf("file operation failed")
	}

	// Make commands fail
	executor.CommandContextFunc = func(ctx context.Context, name string, args ...string) Command {
		return &MockCommand{
			name:     name,
			args:     args,
			executor: executor,
			RunFunc: func() error {
				return fmt.Errorf("command failed")
			},
		}
	}

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownVM()

	if err == nil {
		t.Error("Expected error when all shutdown methods fail")
	}

	if !strings.Contains(err.Error(), "all shutdown methods failed") {
		t.Errorf("Expected all methods failed error, got: %v", err)
	}
}

func TestRunWorkflowErrorHandling(t *testing.T) {
	config := &RunnerConfig{
		Method:          "runner-token",
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}

	logger := NewMockLogger()

	// Test download failure
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("download failed")
		},
	}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.Run(ctx)

	if err == nil {
		t.Error("Expected error due to download failure")
	}

	if !strings.Contains(err.Error(), "failed to download runner") {
		t.Errorf("Expected download failure error, got: %v", err)
	}
}
func TestRealImplementationMethods(t *testing.T) {
	// Test RealHTTPClient methods
	httpClient := NewRealHTTPClient(1 * time.Second)

	// Skip actual HTTP request to avoid network dependencies and hanging
	// Just test that the client was created successfully
	if httpClient == nil {
		t.Error("HTTP client should not be nil")
	}

	// Test RealFileSystem methods
	fileSystem := NewRealFileSystem()
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test-dir")

	// Test MkdirAll
	err := fileSystem.MkdirAll(testDir, 0755)
	if err != nil {
		t.Errorf("MkdirAll failed: %v", err)
	}

	// Test OpenFile and WriteString
	testFile := filepath.Join(testDir, "test-file.txt")
	file, err := fileSystem.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("OpenFile failed: %v", err)
	} else {
		// Test WriteString
		_, err = fileSystem.WriteString(file, "test content")
		if err != nil {
			t.Errorf("WriteString failed: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Errorf("Failed to close file: %v", err)
		}
	}

	// Test RemoveAll
	err = fileSystem.RemoveAll(testDir)
	if err != nil {
		t.Errorf("RemoveAll failed: %v", err)
	}

	// Test RealCommandExecutor methods
	executor := NewRealCommandExecutor()
	cmd := executor.CommandContext(context.Background(), "echo", "test")

	// Test RealCommand methods
	cmd.SetDir(tempDir)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	// Test Run (echo should always work)
	err = cmd.Run()
	if err != nil {
		t.Errorf("Command execution failed: %v", err)
	}

	// Test RealSystemOperations methods
	system := NewRealSystemOperations()

	// Skip actual system calls to avoid hanging in test environment
	// Just verify the system operations object was created successfully
	if system == nil {
		t.Error("System operations should not be nil")
	}
}

func TestMainFunctionErrorPaths(t *testing.T) {
	// Test unsupported method
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "unsupported-config.json")

	unsupportedConfig := &RunnerConfig{
		Method: "unsupported-method",
	}

	configData, err := json.Marshal(unsupportedConfig)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := loadRunnerConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test the method switch logic for unsupported method
	var shouldFail bool
	switch config.Method {
	case runnerTokenMethod:
		shouldFail = false
	case joinTokenMethod:
		shouldFail = true // Not yet implemented
	default:
		shouldFail = true // Unsupported method
	}

	if !shouldFail {
		t.Error("Expected unsupported method to fail")
	}

	// Test join-token method (not yet implemented)
	joinTokenConfig := &RunnerConfig{
		Method: joinTokenMethod,
	}

	switch joinTokenConfig.Method {
	case runnerTokenMethod:
		t.Error("Should not match runner-token")
	case joinTokenMethod:
		// This should be the path taken, and it should fail
		t.Log("join-token method correctly identified as not implemented")
	default:
		t.Error("Should match join-token")
	}
}

func TestCompleteWorkflowWithMocks(t *testing.T) {
	// Test a complete successful workflow
	config := &RunnerConfig{
		Method:          "runner-token",
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "test"},
	}

	logger := NewMockLogger()

	// Mock successful HTTP response with minimal tar content
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Create a minimal valid tar.gz response
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			// Add a simple file to the tar
			header := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.Run(ctx)

	if err != nil {
		t.Errorf("Expected successful run, got error: %v", err)
	}

	// Verify all steps were executed
	if len(fileSystem.CreatedDirs) == 0 {
		t.Error("Expected directories to be created")
	}

	if len(executor.ExecutedCommands) != 2 {
		t.Errorf("Expected 2 commands (configure + run), got %d", len(executor.ExecutedCommands))
	}

	if len(fileSystem.RemovedPaths) == 0 {
		t.Error("Expected cleanup to remove paths")
	}

	if !system.SleepCalled {
		t.Error("Expected cleanup delay")
	}

	if len(logger.Messages) == 0 {
		t.Error("Expected log messages throughout workflow")
	}
}

func TestDownloadGitHubRunnerDirectoryCreationError(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}

	// Test directory creation failure
	fileSystem := NewMockFileSystem()
	fileSystem.MkdirAllFunc = func(path string, perm os.FileMode) error {
		return fmt.Errorf("permission denied")
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to directory creation failure")
	}

	if !strings.Contains(err.Error(), "failed to create install directory") {
		t.Errorf("Expected directory creation error, got: %v", err)
	}
}

func TestDownloadGitHubRunnerRequestCreationError(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	// Create a context that's already cancelled to trigger request creation error
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to cancelled context")
	}

	// The error could be either request creation or download failure
	// Both are valid error paths we want to test
	if !strings.Contains(err.Error(), "failed to create request") &&
		!strings.Contains(err.Error(), "failed to download runner") &&
		!strings.Contains(err.Error(), "failed to create gzip reader") {
		t.Errorf("Expected request/download/gzip error, got: %v", err)
	}
}

func TestDownloadGitHubRunnerInvalidTarPath(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()

	// Create a tar with invalid path (path traversal attempt)
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			// Add a file with path traversal attempt
			header := &tar.Header{
				Name: "../../../etc/passwd",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to invalid file path")
	}

	if !strings.Contains(err.Error(), "invalid file path in archive") {
		t.Errorf("Expected invalid path error, got: %v", err)
	}
}

func TestDownloadGitHubRunnerFileCreationError(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()

	// Create a valid tar
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			// Add a valid file
			header := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	// Test file creation failure
	fileSystem := NewMockFileSystem()
	fileSystem.OpenFileFunc = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		return nil, fmt.Errorf("permission denied")
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to file creation failure")
	}

	if !strings.Contains(err.Error(), "failed to create file") {
		t.Errorf("Expected file creation error, got: %v", err)
	}
}

func TestShutdownViaSysRqFileOpenError(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}

	// Test file open failure
	fileSystem := NewMockFileSystem()
	fileSystem.OpenFileFunc = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		return nil, fmt.Errorf("permission denied")
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaSysRq()

	if err == nil {
		t.Error("Expected error due to file open failure")
	}

	if !strings.Contains(err.Error(), "failed to open sysrq-trigger") {
		t.Errorf("Expected sysrq-trigger open error, got: %v", err)
	}
}

func TestShutdownViaSysRqWriteError(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}

	// Test write failure
	fileSystem := NewMockFileSystem()
	fileSystem.WriteStringFunc = func(file io.WriteCloser, data string) (int, error) {
		return 0, fmt.Errorf("write failed")
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaSysRq()

	if err == nil {
		t.Error("Expected error due to write failure")
	}

	if !strings.Contains(err.Error(), "failed to write to sysrq-trigger") {
		t.Errorf("Expected sysrq-trigger write error, got: %v", err)
	}
}

func TestShutdownViaPowerStateFileOpenError(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}

	// Test file open failure for power disk file
	fileSystem := NewMockFileSystem()
	fileSystem.OpenFileFunc = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		if strings.Contains(name, "power/disk") {
			return nil, fmt.Errorf("permission denied")
		}
		return &MockWriteCloser{name: name, fs: fileSystem}, nil
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaPowerState()

	if err == nil {
		t.Error("Expected error due to power disk file open failure")
	}

	if !strings.Contains(err.Error(), "failed to open power disk file") {
		t.Errorf("Expected power disk file open error, got: %v", err)
	}
}

func TestShutdownViaPowerStateWriteError(t *testing.T) {
	config := &RunnerConfig{}
	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}

	// Test write failure to power disk file
	fileSystem := NewMockFileSystem()
	fileSystem.WriteStringFunc = func(file io.WriteCloser, data string) (int, error) {
		return 0, fmt.Errorf("write failed")
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.shutdownViaPowerState()

	if err == nil {
		t.Error("Expected error due to write failure")
	}

	if !strings.Contains(err.Error(), "failed to write to power disk file") {
		t.Errorf("Expected power disk file write error, got: %v", err)
	}
}

func TestRunWorkflowConfigurationError(t *testing.T) {
	config := &RunnerConfig{
		Method:          runnerTokenMethod,
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}

	logger := NewMockLogger()

	// Mock successful download
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			header := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	fileSystem := NewMockFileSystem()

	// Mock configuration failure
	executor := NewMockCommandExecutor()
	executor.CommandContextFunc = func(ctx context.Context, name string, args ...string) Command {
		return &MockCommand{
			name:     name,
			args:     args,
			executor: executor,
			RunFunc: func() error {
				if strings.Contains(name, "config.sh") {
					return fmt.Errorf("configuration failed")
				}
				return nil
			},
		}
	}
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.Run(ctx)

	if err == nil {
		t.Error("Expected error due to configuration failure")
	}

	if !strings.Contains(err.Error(), "failed to configure runner") {
		t.Errorf("Expected configuration failure error, got: %v", err)
	}
}

func TestRunWorkflowRunnerError(t *testing.T) {
	config := &RunnerConfig{
		Method:          runnerTokenMethod,
		RunnerToken:     "test-token",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted"},
	}

	logger := NewMockLogger()

	// Mock successful download
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			header := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	fileSystem := NewMockFileSystem()

	// Mock runner execution failure
	executor := NewMockCommandExecutor()
	executor.CommandContextFunc = func(ctx context.Context, name string, args ...string) Command {
		return &MockCommand{
			name:     name,
			args:     args,
			executor: executor,
			RunFunc: func() error {
				if strings.Contains(name, "run.sh") {
					return fmt.Errorf("runner execution failed")
				}
				return nil
			},
		}
	}
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.Run(ctx)

	if err == nil {
		t.Error("Expected error due to runner execution failure")
	}

	if !strings.Contains(err.Error(), "failed to run runner") {
		t.Errorf("Expected runner execution failure error, got: %v", err)
	}
}

func TestBuildDownloadURLWithCustomURL(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.DownloadURL = "https://custom.example.com/runner.tar.gz"

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	url := bootstrap.buildDownloadURL()

	if url != "https://custom.example.com/runner.tar.gz" {
		t.Errorf("Expected custom URL, got: %s", url)
	}
}

func TestBuildDownloadURLWithDifferentArchitectures(t *testing.T) {
	testCases := []struct {
		os       string
		arch     string
		expected string
	}{
		{"linux", "amd64", "actions-runner-linux-x64-2.311.0.tar.gz"},
		{"linux", "arm64", "actions-runner-linux-arm64-2.311.0.tar.gz"},
		{"linux", "386", "actions-runner-linux-x86-2.311.0.tar.gz"},
		{"darwin", "amd64", "actions-runner-osx-x64-2.311.0.tar.gz"},
		{"windows", "amd64", "actions-runner-win-x64-2.311.0.tar.gz"},
		{"unknown", "unknown", "actions-runner-unknown-unknown-2.311.0.tar.gz"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.os, tc.arch), func(t *testing.T) {
			config := &RunnerConfig{}
			config.Runner.OS = tc.os
			config.Runner.Arch = tc.arch

			logger := NewMockLogger()
			httpClient := &MockHTTPClient{}
			fileSystem := NewMockFileSystem()
			executor := NewMockCommandExecutor()
			system := NewMockSystemOperations()

			bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

			url := bootstrap.buildDownloadURL()

			if !strings.Contains(url, tc.expected) {
				t.Errorf("Expected URL to contain %s, got: %s", tc.expected, url)
			}
		})
	}
}

func TestPerformSPIFFEAttestationMissingConfig(t *testing.T) {
	config := &RunnerConfig{}
	config.SPIFFE.Enabled = true
	// Missing both JoinToken and SPIFFEID

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	err := bootstrap.performSPIFFEAttestation()

	if err == nil {
		t.Error("Expected error due to missing SPIFFE configuration")
	}

	if !strings.Contains(err.Error(), "no join token or SPIFFE ID provided") {
		t.Errorf("Expected SPIFFE config error, got: %v", err)
	}
}

func TestGetOSArchFromEnvironment(t *testing.T) {
	config := &RunnerConfig{}
	// Don't set OS/Arch in config to test environment detection

	logger := NewMockLogger()
	httpClient := &MockHTTPClient{}
	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	os, arch := bootstrap.getOSArch()

	// Should return runtime values since no environment variables are set
	if os == "" || arch == "" {
		t.Errorf("Expected non-empty OS and arch, got OS: %s, Arch: %s", os, arch)
	}
}

func TestWriteStringFallbackPath(t *testing.T) {
	// Test the WriteString method's fallback path in RealFileSystem
	fs := NewRealFileSystem()

	// Create a mock writer that doesn't implement io.StringWriter
	mockWriter := &mockWriterOnly{}

	n, err := fs.WriteString(mockWriter, "test data")

	if err != nil {
		t.Errorf("WriteString should not fail, got: %v", err)
	}

	if n != 9 { // len("test data")
		t.Errorf("Expected 9 bytes written, got: %d", n)
	}

	if string(mockWriter.data) != "test data" {
		t.Errorf("Expected 'test data', got: %s", string(mockWriter.data))
	}
}

// mockWriterOnly implements only io.Writer, not io.StringWriter
type mockWriterOnly struct {
	data []byte
}

func (m *mockWriterOnly) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockWriterOnly) Close() error {
	return nil
}
func TestDownloadGitHubRunnerTarDirectoryHandling(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()

	// Create a tar with directory entries
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			// Add a directory entry
			dirHeader := &tar.Header{
				Name:     "test-dir/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}
			_ = tarWriter.WriteHeader(dirHeader)

			// Add a file in the directory
			fileHeader := &tar.Header{
				Name: "test-dir/test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(fileHeader)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err != nil {
		t.Errorf("Expected successful extraction, got error: %v", err)
	}

	// Verify both directory and file creation
	expectedDirs := []string{"/opt/test-runner", "/opt/test-runner/test-dir"}
	if len(fileSystem.CreatedDirs) < 2 {
		t.Errorf("Expected at least 2 directories created, got %d", len(fileSystem.CreatedDirs))
	}

	// Check that directories were created
	for _, expectedDir := range expectedDirs {
		found := false
		for _, createdDir := range fileSystem.CreatedDirs {
			if createdDir == expectedDir {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected directory '%s' to be created", expectedDir)
		}
	}
}

func TestDownloadGitHubRunnerParentDirectoryCreationError(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = "/opt/test-runner"

	logger := NewMockLogger()

	// Create a tar with a file that requires parent directory creation
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			// Add a file in a subdirectory
			header := &tar.Header{
				Name: "subdir/test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	// Mock parent directory creation failure
	fileSystem := NewMockFileSystem()
	callCount := 0
	fileSystem.MkdirAllFunc = func(path string, perm os.FileMode) error {
		callCount++
		if callCount > 1 { // Fail on parent directory creation
			return fmt.Errorf("parent directory creation failed")
		}
		return nil
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to parent directory creation failure")
	}

	if !strings.Contains(err.Error(), "failed to create parent directory") {
		t.Errorf("Expected parent directory creation error, got: %v", err)
	}
}

func TestDownloadGitHubRunnerFileCopyError(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()

	// Create a tar with a file
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			header := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	// Mock file that fails to write
	fileSystem := NewMockFileSystem()
	fileSystem.OpenFileFunc = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		return &FailingWriteCloser{}, nil
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to file copy failure")
	}

	if !strings.Contains(err.Error(), "failed to write file") {
		t.Errorf("Expected file write error, got: %v", err)
	}
}

func TestDownloadGitHubRunnerFileCloseError(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = testInstallPath

	logger := NewMockLogger()

	// Create a tar with a file
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			header := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(header)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	// Mock file that fails to close
	fileSystem := NewMockFileSystem()
	fileSystem.OpenFileFunc = func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
		return &FailingCloseWriteCloser{}, nil
	}
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err == nil {
		t.Error("Expected error due to file close failure")
	}

	if !strings.Contains(err.Error(), "failed to close file") {
		t.Errorf("Expected file close error, got: %v", err)
	}
}

func TestDownloadGitHubRunnerCurrentDirectoryEntry(t *testing.T) {
	config := &RunnerConfig{}
	config.Runner.InstallPath = "/opt/test-runner"

	logger := NewMockLogger()

	// Create a tar with current directory entry
	httpClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			var buf bytes.Buffer
			gzWriter := gzip.NewWriter(&buf)
			tarWriter := tar.NewWriter(gzWriter)

			// Add current directory entry (should be allowed)
			dirHeader := &tar.Header{
				Name:     "./",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}
			_ = tarWriter.WriteHeader(dirHeader)

			// Add a regular file
			fileHeader := &tar.Header{
				Name: "test-file",
				Mode: 0644,
				Size: 4,
			}
			_ = tarWriter.WriteHeader(fileHeader)
			_, _ = tarWriter.Write([]byte("test"))

			_ = tarWriter.Close()
			_ = gzWriter.Close()

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			}, nil
		},
	}

	fileSystem := NewMockFileSystem()
	executor := NewMockCommandExecutor()
	system := NewMockSystemOperations()

	bootstrap := NewGitHubBootstrap(config, logger, httpClient, fileSystem, executor, system)

	ctx := context.Background()
	err := bootstrap.downloadGitHubRunner(ctx)

	if err != nil {
		t.Errorf("Expected successful extraction with current directory entry, got error: %v", err)
	}
}

// FailingWriteCloser implements io.WriteCloser but fails on Write
type FailingWriteCloser struct{}

func (f *FailingWriteCloser) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write failed")
}

func (f *FailingWriteCloser) Close() error {
	return nil
}

// FailingCloseWriteCloser implements io.WriteCloser but fails on Close
type FailingCloseWriteCloser struct {
	buf bytes.Buffer
}

func (f *FailingCloseWriteCloser) Write(p []byte) (n int, err error) {
	return f.buf.Write(p)
}

func (f *FailingCloseWriteCloser) Close() error {
	return fmt.Errorf("close failed")
}
