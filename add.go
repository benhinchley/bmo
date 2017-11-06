package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	git "gopkg.in/src-d/go-git.v4"

	"github.com/benhinchley/cmd"
)

const addDesc = "Add file to the workspace stage."
const addHelp = `
--- TODO ---
`

type addCommand struct{}

func (cmd *addCommand) Name() string           { return "add" }
func (cmd *addCommand) Args() string           { return "<workspace> <repo/file>..." }
func (cmd *addCommand) Desc() string           { return strings.TrimSpace(addDesc) }
func (cmd *addCommand) Help() string           { return strings.TrimSpace(addHelp) }
func (cmd *addCommand) Register(*flag.FlagSet) {}

func (cmd *addCommand) Run(ctx cmd.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("not enough arguments provided")
	}

	workspace := args[0]
	s, err := ctx.(*context).Config.GetSection(fmt.Sprintf("workspace.%s", workspace))
	if err != nil {
		return fmt.Errorf("%s does not exist: %v", workspace, err)
	}

	repos := s.Key("repos").Strings(",")
	files := strings.Split(strings.Join(args[1:], ","), ",")
	for _, f := range files {
		a := strings.SplitN(f, "/", 2)
		repo := getRepo(repos, a[0])
		if repo == "" {
			return fmt.Errorf("%s not a part of workspace: %v", repo, err)
		}
		file := a[1]

		r, err := git.PlainOpen(repo)
		if err != nil {
			return fmt.Errorf("unable to open repo: %v", err)
		}
		wt, err := r.Worktree()
		if err != nil {
			return fmt.Errorf("unable to get repository worktree: %v", err)
		}

		if _, err := wt.Add(file); err != nil {
			return fmt.Errorf("unable to stage \"%s\": %v", file, err)
		}
	}

	return nil
}

func getRepo(repos []string, repo string) string {
	for _, r := range repos {
		if filepath.Base(r) == repo {
			return r
		}
	}
	return ""
}
