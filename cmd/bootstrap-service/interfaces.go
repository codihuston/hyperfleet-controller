package main

import (
	"context"
	"io"
	"net/http"
	"os"
)

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FileSystem interface for file operations
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
	WriteString(file io.WriteCloser, data string) (int, error)
}

// CommandExecutor interface for executing commands
type CommandExecutor interface {
	CommandContext(ctx context.Context, name string, args ...string) Command
}

// Command interface for command execution
type Command interface {
	Run() error
	SetDir(dir string)
	SetStdout(stdout io.Writer)
	SetStderr(stderr io.Writer)
}

// SystemOperations interface for system-level operations
type SystemOperations interface {
	Sync()
	Reboot(cmd int) error
	Sleep(duration int)
}

// Logger interface for logging operations
type Logger interface {
	Printf(format string, v ...interface{})
}
