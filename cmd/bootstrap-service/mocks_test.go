package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// MockHTTPClient implements HTTPClient for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
}

// MockFileSystem implements FileSystem for testing
type MockFileSystem struct {
	MkdirAllFunc    func(path string, perm os.FileMode) error
	RemoveAllFunc   func(path string) error
	OpenFileFunc    func(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
	WriteStringFunc func(file io.WriteCloser, data string) (int, error)

	CreatedDirs  []string
	RemovedPaths []string
	OpenedFiles  []string
	WrittenData  map[string]string
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		CreatedDirs:  make([]string, 0),
		RemovedPaths: make([]string, 0),
		OpenedFiles:  make([]string, 0),
		WrittenData:  make(map[string]string),
	}
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	m.CreatedDirs = append(m.CreatedDirs, path)
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path, perm)
	}
	return nil
}

func (m *MockFileSystem) RemoveAll(path string) error {
	m.RemovedPaths = append(m.RemovedPaths, path)
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(path)
	}
	return nil
}

func (m *MockFileSystem) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	m.OpenedFiles = append(m.OpenedFiles, name)
	if m.OpenFileFunc != nil {
		return m.OpenFileFunc(name, flag, perm)
	}
	return &MockWriteCloser{name: name, fs: m}, nil
}

func (m *MockFileSystem) WriteString(file io.WriteCloser, data string) (int, error) {
	// Handle the case where we're writing to a mock file
	if mockFile, ok := file.(*MockWriteCloser); ok {
		m.WrittenData[mockFile.name] = data
		mockFile.buf.WriteString(data)
	} else if len(m.OpenedFiles) > 0 {
		// For other cases, try to identify the file by checking opened files
		// This is a fallback for when the file interface doesn't match our mock
		lastFile := m.OpenedFiles[len(m.OpenedFiles)-1]
		m.WrittenData[lastFile] = data
	}
	if m.WriteStringFunc != nil {
		return m.WriteStringFunc(file, data)
	}
	return len(data), nil
}

// MockWriteCloser implements io.WriteCloser for testing
type MockWriteCloser struct {
	name string
	fs   *MockFileSystem
	buf  bytes.Buffer
}

func (m *MockWriteCloser) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *MockWriteCloser) Close() error {
	m.fs.WrittenData[m.name] = m.buf.String()
	return nil
}

// MockCommandExecutor implements CommandExecutor for testing
type MockCommandExecutor struct {
	CommandContextFunc func(ctx context.Context, name string, args ...string) Command
	ExecutedCommands   []MockExecutedCommand
}

type MockExecutedCommand struct {
	Name string
	Args []string
	Dir  string
}

func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		ExecutedCommands: make([]MockExecutedCommand, 0),
	}
}

func (m *MockCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) Command {
	if m.CommandContextFunc != nil {
		return m.CommandContextFunc(ctx, name, args...)
	}
	return &MockCommand{
		name:     name,
		args:     args,
		executor: m,
	}
}

// MockCommand implements Command for testing
type MockCommand struct {
	name     string
	args     []string
	dir      string
	executor *MockCommandExecutor
	RunFunc  func() error
}

func (m *MockCommand) Run() error {
	if m.executor != nil {
		m.executor.ExecutedCommands = append(m.executor.ExecutedCommands, MockExecutedCommand{
			Name: m.name,
			Args: m.args,
			Dir:  m.dir,
		})
	}
	if m.RunFunc != nil {
		return m.RunFunc()
	}
	return nil
}

func (m *MockCommand) SetDir(dir string) {
	m.dir = dir
}

func (m *MockCommand) SetStdout(stdout io.Writer) {
	// Mock implementation - could store for verification if needed
}

func (m *MockCommand) SetStderr(stderr io.Writer) {
	// Mock implementation - could store for verification if needed
}

// MockSystemOperations implements SystemOperations for testing
type MockSystemOperations struct {
	SyncFunc   func()
	RebootFunc func(cmd int) error
	SleepFunc  func(duration int)

	SyncCalled    bool
	RebootCalled  bool
	RebootCmd     int
	SleepCalled   bool
	SleepDuration int
}

func NewMockSystemOperations() *MockSystemOperations {
	return &MockSystemOperations{}
}

func (m *MockSystemOperations) Sync() {
	m.SyncCalled = true
	if m.SyncFunc != nil {
		m.SyncFunc()
	}
}

func (m *MockSystemOperations) Reboot(cmd int) error {
	m.RebootCalled = true
	m.RebootCmd = cmd
	if m.RebootFunc != nil {
		return m.RebootFunc(cmd)
	}
	return nil
}

func (m *MockSystemOperations) Sleep(duration int) {
	m.SleepCalled = true
	m.SleepDuration = duration
	if m.SleepFunc != nil {
		m.SleepFunc(duration)
	}
}

// MockLogger implements Logger for testing
type MockLogger struct {
	PrintfFunc func(format string, v ...interface{})
	Messages   []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: make([]string, 0),
	}
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	m.Messages = append(m.Messages, message)
	if m.PrintfFunc != nil {
		m.PrintfFunc(format, v...)
	}
}
