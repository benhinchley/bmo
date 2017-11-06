package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/benhinchley/cmd"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

const logDesc = "Show the commit logs for the workspace."
const logHelp = `
--- TODO ---
`

type logCommand struct {
	oneline bool
}

func (cmd *logCommand) Name() string { return "log" }
func (cmd *logCommand) Args() string { return "[-oneline] <workspace>" }
func (cmd *logCommand) Desc() string { return strings.TrimSpace(logDesc) }
func (cmd *logCommand) Help() string { return strings.TrimSpace(logHelp) }
func (cmd *logCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.oneline, "oneline", false, "mimics \"git log --oneline\"")
}

func (cmd *logCommand) Run(ctx cmd.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("not enough arguments provided")
	}

	workspace := args[0]
	s, err := ctx.(*context).Config.GetSection(fmt.Sprintf("workspace.%s", workspace))
	if err != nil {
		return fmt.Errorf("%s does not exist: %v", workspace, err)
	}

	errChan := make(chan error)
	repos := s.Key("repos").Strings(",")
	var wg sync.WaitGroup
	for _, repo := range repos {
		r, err := git.PlainOpen(repo)
		if err != nil {
			return fmt.Errorf("unable to open repo: %v", err)
		}
		ref, err := r.Head()
		if err != nil {
			return fmt.Errorf("unable to get ref HEAD: %v", err)
		}
		ci, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			return fmt.Errorf("unable to get commit iter: %v", err)
		}

		wg.Add(1)
		go func(repo string, ci object.CommitIter) {
			defer wg.Done()
			if err := ci.ForEach(func(c *object.Commit) error {
				msg := fmt.Sprintf("[%s] %s %s", filepath.Base(repo), c.Hash.String()[:7],
					strings.TrimSpace(strings.Split(c.Message, "\n")[0]))
				if !cmd.oneline {
					msg = fmt.Sprintf("commit %s\nRepository: %s\nAuthor:     %s\nDate:       %s\n\n%s\n",
						c.Hash, filepath.Base(repo), c.Author.String(),
						c.Author.When.Format(object.DateFormat), indent(c.Message))
					msg = strings.TrimSpace(msg) + "\n"
				}

				ctx.Stdout().Println(msg)
				return nil
			}); err != nil {
				errChan <- err
			}
		}(repo, ci)
	}
	wg.Wait()

	close(errChan)
	return <-errChan
}

func indent(t string) string {
	var output []string
	for _, line := range strings.Split(t, "\n") {
		if len(line) != 0 {
			line = "    " + line
		}

		output = append(output, line)
	}

	return strings.Join(output, "\n")
}
