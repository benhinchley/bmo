package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	homedir "github.com/mitchellh/go-homedir"

	"github.com/go-ini/ini"
)

type command interface {
	Name() string
	Args() string
	Desc() string
	Help() string
	Register(*flag.FlagSet)
	Run(*context, []string) error
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bmo: failed to get working directory: %v\n", err)
		os.Exit(1)
	}
	c := &config{
		Args:       os.Args,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		WorkingDir: wd,
		Env:        os.Environ(),
	}
	os.Exit(c.Run())
}

type context struct {
	WorkingDir     string
	Config         *ini.File
	Out, Err       *log.Logger
	RawOut, RawErr io.Writer
}

type config struct {
	WorkingDir     string
	Args           []string
	Env            []string
	Stdout, Stderr io.Writer
}

func (c *config) Run() int {
	cmds := []command{
		&cloneCommand{},
		&logCommand{},
	}

	stdout := log.New(c.Stdout, "", 0)
	stderr := log.New(c.Stderr, "", 0)

	usage := func() {
		stderr.Println("bmo is a tool for managing many repositories as if they were a monorepo.")
		stderr.Println("")
		stderr.Println("Usage: bmo <command>")
		stderr.Println("")
		stderr.Println("Commands:")
		stderr.Println()
		w := tabwriter.NewWriter(c.Stderr, 0, 0, 2, ' ', 0)
		for _, cmd := range cmds {
			fmt.Fprintf(w, "\t%s\t%s\n", cmd.Name(), cmd.Desc())
		}
		w.Flush()
		stderr.Println()
		stderr.Println("Use \"bmo help [command]\" for more information about a command.")
	}

	help, command, exit := parseArgs(os.Args[1:])
	if exit {
		usage()
		return 1
	}

	for _, cmd := range cmds {
		if cmd.Name() == command {
			fs := flag.NewFlagSet(command, flag.ContinueOnError)
			fs.SetOutput(c.Stderr)
			cmd.Register(fs)

			setCommandUsage(stderr, fs, cmd)

			if help {
				fs.Usage()
				return 1
			}

			if err := fs.Parse(c.Args[2:]); err != nil {
				return 1
			}

			cp, err := homedir.Expand("~/.bmoconfig")
			if err != nil {
				stderr.Printf("bmo: unable to expand path to config file: %v\n", err)
				os.Exit(1)
			}
			f, err := os.Open(cp)
			if err != nil {
				f, err = os.Create(cp)
				if err != nil {
					stderr.Printf("bmo: unable to create ~/.bmoconfig: %v\n", err)
					return 1
				}
			}

			cfg, err := ini.InsensitiveLoad(f)
			if err != nil {
				stderr.Printf("bmo: unable to load config: %v\n", err)
				return 1
			}

			if err := cmd.Run(&context{
				WorkingDir: c.WorkingDir,
				Config:     cfg,
				Out:        stdout,
				Err:        stderr,
				RawOut:     c.Stdout,
				RawErr:     c.Stderr,
			}, fs.Args()); err != nil {
				stderr.Printf("bmo: %v\n", err)
				return 1
			}

			f, err = os.OpenFile(cp, os.O_TRUNC|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				stderr.Printf("bmo: unable to open config file: %v\n", err)
				return 1
			}
			if _, err := cfg.WriteTo(f); err != nil {
				stderr.Printf("bmo: unable to save config file: %v\n", err)
				return 1
			}

			return 0
		}
	}

	stderr.Printf("bmo: %s: no such command\n", command)
	usage()
	return 1
}

func parseArgs(args []string) (usage bool, cmd string, exit bool) {
	isHelp := func(arg string) bool {
		return strings.Contains(strings.ToLower(arg), "help") || strings.ToLower(arg) == "-h"
	}

	switch len(args) {
	case 0:
		exit = true
	case 1:
		if isHelp(args[0]) {
			exit = true
		}
		cmd = args[0]
	default:
		if isHelp(args[0]) {
			cmd = args[1]
			usage = true
		} else {
			cmd = args[0]
		}
	}

	return usage, cmd, exit
}

func setCommandUsage(l *log.Logger, fs *flag.FlagSet, cmd command) {
	var (
		flags bool
		fb    bytes.Buffer
		fw    = tabwriter.NewWriter(&fb, 0, 4, 2, ' ', 0)
	)

	fs.VisitAll(func(f *flag.Flag) {
		flags = true
		dv := f.DefValue
		if dv == "" {
			dv = "<none>"
		}
		fmt.Fprintf(fw, "\t-%s\t%s (default: %s)\n", f.Name, f.Usage, dv)
	})
	fw.Flush()

	fs.Usage = func() {
		l.Printf("Usage: bmo %s %s\n", cmd.Name(), cmd.Args())
		l.Println("")
		l.Println(strings.TrimSpace(cmd.Help()))
		l.Println("")
		if flags {
			l.Println("Flags:")
			l.Println("")
			l.Println(fb.String())
		}
	}
}
