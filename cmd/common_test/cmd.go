package ct

import (
	"bytes"
	"os/exec"
	"syscall"

	"gopkg.in/tomb.v2"
)

type Cmd struct {
	Cmd *exec.Cmd
	Out *bytes.Buffer
	t   tomb.Tomb
}

func Exec(cmdName string, args ...string) (*Cmd, error) {
	cmd := exec.Command(cmdName, args...)
	out := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = out

	c := &Cmd{
		Cmd: cmd,
		Out: out,
	}

	if err := c.Cmd.Start(); err != nil {
		return c, err
	}

	c.t.Go(c.Cmd.Wait)
	return c, nil
}

func (c *Cmd) Stop() error {
	if !c.Alive() {
		return nil
	}
	if err := c.Cmd.Process.Kill(); err != nil {
		return err
	}
	c.t.Wait()
	return nil
}

func (c *Cmd) Wait() error {
	if c.t.Alive() {
		return c.t.Wait()
	}
	return c.t.Err()
}

func (c *Cmd) Alive() bool {
	return c.t.Alive()
}

func (c *Cmd) ExitStatus() (int, error) {
	err := c.t.Err()
	return ExitStatus(err), err
}

func ExecSync(cmdName string, args ...string) (*Cmd, error) {
	cmd := exec.Command(cmdName, args...)
	c := &Cmd{
		Cmd: cmd,
	}

	var out []byte
	c.t.Go(func() error {
		var err error
		out, err = cmd.CombinedOutput()
		return err
	})

	err := c.Wait()
	c.Out = bytes.NewBuffer(out)
	return c, err
}

func ExitStatus(err error) int {
	exitStatus := 0
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitStatus = status.ExitStatus()
			}
		}
	}
	return exitStatus
}

func Build() error {
	_, err := ExecSync("go", "build")
	return err
}
