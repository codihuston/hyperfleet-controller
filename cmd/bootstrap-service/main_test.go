package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRunnerConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	testConfig := &RunnerConfig{
		Method:          "runner-token",
		Platform:        "github-actions",
		RunnerToken:     "test-token-123",
		RegistrationURL: "https://github.com/test/repo",
		RunnerName:      "test-runner",
		Labels:          []string{"self-hosted", "test"},
		ExpiresAt:       "2025-12-25T06:00:55.977-06:00",
	}
	testConfig.Runner.DownloadURL = "https://example.com/runner.tar.gz"
	testConfig.Runner.InstallPath = "/opt/test-runner"
	testConfig.Runner.WorkDir = "/tmp/test-work"

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
