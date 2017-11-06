package main

import (
	"bytes"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/benhinchley/cmd"

	git "gopkg.in/src-d/go-git.v4"
)

const statusDesc = "Show the working tree status for the workspace."
const statusHelp = `
--- TODO ---
`

type statusCommand struct {
	short bool
}

func (cmd *statusCommand) Name() string { return "status" }
func (cmd *statusCommand) Args() string { return "<workspace>" }
func (cmd *statusCommand) Desc() string { return strings.TrimSpace(statusDesc) }
func (cmd *statusCommand) Help() string { return strings.TrimSpace(statusHelp) }
func (cmd *statusCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.short, "short", false, "Give the output in the short-format.")
}

func (cmd *statusCommand) Run(ctx cmd.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("not enough arguments provided")
	}

	workspace := args[0]
	s, err := ctx.(*context).Config.GetSection(fmt.Sprintf("workspace.%s", workspace))
	if err != nil {
		return fmt.Errorf("%s does not exist: %v", workspace, err)
	}

	repos := s.Key("repos").Strings(",")
	var wg sync.WaitGroup
	for _, repo := range repos {
		r, err := git.PlainOpen(repo)
		if err != nil {
			return fmt.Errorf("unable to open repo \"%s\": %v", filepath.Base(repo), err)
		}
		wt, err := r.Worktree()
		if err != nil {
			return fmt.Errorf("unable to get repository worktree: %v", err)
		}
		st, err := wt.Status()
		if err != nil {
			return fmt.Errorf("unable to get repository worktree status: %v", err)
		}

		wg.Add(1)
		go func(repo string, ws map[string]*git.FileStatus) {
			defer wg.Done()

			if cmd.short {
				if len(ws) == 0 {
					ctx.Stdout().Printf("[%s] nothing to commit, working tree clean\n", filepath.Base(repo))
					return
				}

				for file, status := range ws {
					ctx.Stdout().Printf("[%s] %s%s %s", filepath.Base(repo), string(status.Staging), string(status.Worktree), file)
				}
				return
			}

			if len(ws) == 0 {
				ctx.Stdout().Printf("Repository: %s\n", filepath.Base(repo))
				ctx.Stdout().Println("nothing to commit, working tree clean")
				ctx.Stdout().Println("")
				return
			}

			staged := []*entry{}
			unstaged := []*entry{}
			untracked := []*entry{}
			for file, status := range ws {
				switch status.Worktree {
				case git.Untracked:
					untracked = append(untracked, &entry{Filename: file, Status: status.Worktree})
				case git.Modified, git.Added, git.Deleted, git.Renamed, git.Copied:
					unstaged = append(unstaged, &entry{Filename: file, Status: status.Worktree})
				}

				switch status.Staging {
				case git.Modified, git.Added, git.Deleted, git.Renamed, git.Copied:
					staged = append(staged, &entry{Filename: file, Status: status.Staging})
				}
			}

			var sb bytes.Buffer
			sw := tabwriter.NewWriter(&sb, 0, 4, 4, ' ', 0)

			var unsb bytes.Buffer
			unsw := tabwriter.NewWriter(&unsb, 0, 4, 4, ' ', 0)

			var utb bytes.Buffer
			utw := tabwriter.NewWriter(&utb, 0, 4, 4, ' ', 0)

			if len(staged) > 0 {
				for _, e := range staged {
					fmt.Fprintf(sw, "\t%s: %s\n", statusToString(e.Status), e.Filename)
				}
				sw.Flush()
			}

			if len(unstaged) > 0 {
				for _, e := range unstaged {
					fmt.Fprintf(unsw, "\t%s\n", e.Filename)
				}
				unsw.Flush()
			}

			if len(untracked) > 0 {
				for _, e := range untracked {
					fmt.Fprintf(utw, "\t%s\n", e.Filename)
				}
				utw.Flush()
			}

			ctx.Stdout().Printf("Repository: %s\n", filepath.Base(repo))
			if len(staged) > 0 {
				ctx.Stdout().Println("Changes to be committed:")
				ctx.Stdout().Println("")
				ctx.Stdout().Println(sb.String())
			}
			if len(unstaged) > 0 {
				ctx.Stdout().Println("Changes not staged for commit:")
				ctx.Stdout().Println("")
				ctx.Stdout().Println(unsb.String())
			}
			if len(untracked) > 0 {
				ctx.Stdout().Println("Untracked Files:")
				ctx.Stdout().Println("")
				ctx.Stdout().Println(utb.String())
			}
		}(repo, st)
	}
	wg.Wait()

	return nil
}

func statusToString(sc git.StatusCode) string {
	switch sc {
	case git.Unmodified:
		return "unmodified"
	case git.Untracked:
		return "untracked"
	case git.Modified:
		return "modified"
	case git.Added:
		return "added"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Copied:
		return "copied"
	default:
		return ""
	}
}

type entry struct {
	Filename string
	Status   git.StatusCode
}
