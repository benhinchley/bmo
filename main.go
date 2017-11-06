package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/benhinchley/cmd"
	homedir "github.com/mitchellh/go-homedir"

	"github.com/go-ini/ini"
)

func main() {
	p, err := cmd.NewProgram("bmo", "work on lots of repos as if they were a monorepo", nil, []cmd.Command{
		&cloneCommand{},
		&logCommand{},
		&statusCommand{},
		&addCommand{},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := p.ParseArgs(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := p.Run(func(env cmd.Environment, c cmd.Command, args []string) error {
		lout, lerr := env.GetLoggers()
		stdout, stderr := env.GetStdio()

		cp, err := homedir.Expand("~/.bmoconfig")
		if err != nil {
			return fmt.Errorf("unable to expand path to config file: %v", err)
		}
		f, err := os.Open(cp)
		if err != nil {
			f, err = os.Create(cp)
			if err != nil {
				return fmt.Errorf("unable to create ~/.bmoconfig: %v\n", err)
			}
		}

		cfg, err := ini.InsensitiveLoad(f)
		if err != nil {
			return fmt.Errorf("unable to load config: %v\n", err)
		}

		ctx := &context{
			WorkingDir: env.WorkingDir(),
			Config:     cfg,
			RawOut:     stdout,
			RawErr:     stderr,
			out:        lout,
			err:        lerr,
		}

		if err := c.Run(ctx, args); err != nil {
			return fmt.Errorf("%s: %v", c.Name(), err)
		}

		f, err = os.OpenFile(cp, os.O_TRUNC|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return fmt.Errorf("unable to open config file: %v\n", err)
		}
		if _, err := cfg.WriteTo(f); err != nil {
			return fmt.Errorf("unable to save config file: %v\n", err)
		}

		return nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

type context struct {
	WorkingDir     string
	Config         *ini.File
	RawOut, RawErr io.Writer
	out, err       *log.Logger
}

// Implement cmd.Context
func (c *context) Stdout() *log.Logger { return c.out }
func (c *context) Stderr() *log.Logger { return c.err }
