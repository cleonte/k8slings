package cluster

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
)

// This file is the only place in k8slings that runs another program (the
// test TestOnlyExecGoRunsPrograms enforces it). Everything that talks to
// Docker, kind or kubectl goes through a Runner, so that the rules about which
// cluster may be touched live in one package and can be tested with a fake.

// Cmd is one program to run.
type Cmd struct {
	Name string
	Args []string
	// Env is the whole environment of the program. Nil means this process's.
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// Runner runs programs.
type Runner interface {
	Run(ctx context.Context, c Cmd) error
	// Has reports whether a program is on the PATH.
	Has(name string) bool
}

// ExecRunner runs real programs.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, c Cmd) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Env = c.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = c.Stdin, c.Stdout, c.Stderr
	return cmd.Run()
}

func (ExecRunner) Has(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// output runs a program and returns what it printed, stdout and stderr
// together, without the trailing newline.
func output(ctx context.Context, r Runner, env []string, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	err := r.Run(ctx, Cmd{Name: name, Args: args, Env: env, Stdout: &buf, Stderr: &buf})
	return strings.TrimSpace(buf.String()), err
}
