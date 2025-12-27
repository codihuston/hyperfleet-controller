package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// RealHTTPClient implements HTTPClient using the standard http.Client
type RealHTTPClient struct {
	client *http.Client
}

func NewRealHTTPClient(timeout time.Duration) *RealHTTPClient {
	return &RealHTTPClient{
		client: &http.Client{Timeout: timeout},
	}
}

func (c *RealHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// RealFileSystem implements FileSystem using the standard os package
type RealFileSystem struct{}

func NewRealFileSystem() *RealFileSystem {
	return &RealFileSystem{}
}

func (fs *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (fs *RealFileSystem) OpenFile(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	// #nosec G304 - File path is validated by caller, needed for legitimate file operations
	return os.OpenFile(name, flag, perm)
}

func (fs *RealFileSystem) WriteString(file io.WriteCloser, data string) (int, error) {
	if writer, ok := file.(io.StringWriter); ok {
		return writer.WriteString(data)
	}
	return file.Write([]byte(data))
}

// RealCommandExecutor implements CommandExecutor using the standard exec package
type RealCommandExecutor struct{}

func NewRealCommandExecutor() *RealCommandExecutor {
	return &RealCommandExecutor{}
}

func (e *RealCommandExecutor) CommandContext(ctx context.Context, name string, args ...string) Command {
	cmd := exec.CommandContext(ctx, name, args...)
	return &RealCommand{cmd: cmd}
}

// RealCommand implements Command using the standard exec.Cmd
type RealCommand struct {
	cmd *exec.Cmd
}

func (c *RealCommand) Run() error {
	return c.cmd.Run()
}

func (c *RealCommand) SetDir(dir string) {
	c.cmd.Dir = dir
}

func (c *RealCommand) SetStdout(stdout io.Writer) {
	c.cmd.Stdout = stdout
}

func (c *RealCommand) SetStderr(stderr io.Writer) {
	c.cmd.Stderr = stderr
}

// RealSystemOperations implements SystemOperations using syscalls
type RealSystemOperations struct{}

func NewRealSystemOperations() *RealSystemOperations {
	return &RealSystemOperations{}
}

func (s *RealSystemOperations) Sync() {
	syscall.Sync()
}

func (s *RealSystemOperations) Reboot(cmd int) error {
	return syscall.Reboot(cmd)
}

func (s *RealSystemOperations) Sleep(duration int) {
	time.Sleep(time.Duration(duration) * time.Second)
}

// RealLogger implements Logger using the standard log package
type RealLogger struct {
	logger *log.Logger
}

func NewRealLogger(prefix string) *RealLogger {
	return &RealLogger{
		logger: log.New(os.Stdout, prefix, log.LstdFlags),
	}
}

func (l *RealLogger) Printf(format string, v ...interface{}) {
	l.logger.Printf(format, v...)
}
