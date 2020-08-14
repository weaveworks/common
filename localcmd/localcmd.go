package localcmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// LocalCmd runs a command locally
type LocalCmd struct {
	Command    string
	GlobalArgs []string
	Env        []string
}

// LookPath conveniently wraps exec.LookPath(Command)
func (k LocalCmd) LookPath() (string, error) { return exec.LookPath(k.Command) }

// IsPresent returns true if there's a command in the PATH.
func (k LocalCmd) IsPresent() bool {
	_, err := k.LookPath()
	return err == nil
}

// Execute executes command <args> and returns the combined stdout/err output.
func (k LocalCmd) Execute(args ...string) (string, error) {
	cmd := exec.Command(k.Command, append(k.GlobalArgs, args...)...)
	cmd.Env = append(os.Environ(), k.Env...)
	stdout, stderr, err := outputMatrix(cmd)
	if err != nil {
		// error messages output to stdout
		return "", fmt.Errorf("%s\nFull output:\n%s\n%s", trimOutput(stderr), trimOutput(stdout), trimOutput(stderr))
	}
	combined := fmt.Sprintf("%s\n%s", stdout, stderr)
	return trimOutput(combined), nil
}

// ExecuteOutputMatrix executes command <args> and returns stdout and stderr
func (k LocalCmd) ExecuteOutputMatrix(args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(k.Command, append(k.GlobalArgs, args...)...)
	cmd.Env = append(os.Environ(), k.Env...)
	return outputMatrix(cmd)
}

func outputMatrix(cmd *exec.Cmd) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	var wg sync.WaitGroup
	copy := func(dst io.Writer, src io.Reader) {
		defer wg.Done()
		_, _ = io.Copy(dst, src)
	}

	err = cmd.Start()
	if err == nil {
		wg.Add(2)
		go copy(&stdoutBuf, stdoutPipe)
		go copy(&stderrBuf, stderrPipe)
		// we need to wait for all reads to finish before calling cmd.Wait
		wg.Wait()
		err = cmd.Wait()
	}
	stdout, stderr = string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
	return
}

func trimOutput(output string) string {
	return strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(output), "'"), "'")
}
