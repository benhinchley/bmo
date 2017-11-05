package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

// https://stackoverflow.com/a/22312124
var gitURL = regexp.MustCompile(`(?s)((git|ssh|http(s)?)|(git@[\w\.]+))(:(//)?)([\w\.@\:/\-~]+)(\.git)(/)?`)

const cloneDesc = "Clone a repository into a new directory and add to workspace."
const cloneHelp = `
--- TODO ---
`

type cloneCommand struct{}

func (cmd *cloneCommand) Name() string           { return "clone" }
func (cmd *cloneCommand) Args() string           { return "<workspace> <url> [path]" }
func (cmd *cloneCommand) Desc() string           { return strings.TrimSpace(cloneDesc) }
func (cmd *cloneCommand) Help() string           { return strings.TrimSpace(cloneHelp) }
func (cmd *cloneCommand) Register(*flag.FlagSet) {}

func (cmd *cloneCommand) Run(ctx *context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("clone: not enough arguments provided")
	}

	workspace := args[0]
	url := args[1]
	path := filepath.Join(ctx.WorkingDir, filepath.Base(gitURL.FindStringSubmatch(url)[7]))

	if len(args) == 3 {
		path = args[2]
	}

	s := ctx.Config.Section(fmt.Sprintf("workspace.%s", workspace))
	repos := s.Key("repos").Strings(",")

	if contains(repos, path) {
		return fmt.Errorf("clone: %s already exists in %s workspace", url, workspace)
	}

	ep, err := transport.NewEndpoint(url)
	if err != nil {
		return fmt.Errorf("clone: %v", err)
	}

	auth, err := createAuth(ep)
	if err != nil {
		return fmt.Errorf("clone: %v", err)
	}

	opts := &git.CloneOptions{
		URL:      url,
		Auth:     auth,
		Progress: ctx.RawErr,
	}

	if _, err := git.PlainClone(path, false, opts); err != nil {
		switch err {
		case git.ErrRepositoryAlreadyExists:
			return fmt.Errorf("clone: failed to clone %s: %v", url, err)
		default:
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("clone: failed to remove %s: %v", path, err)
			}
		}
	}

	repos = append(repos, path)
	s.Key("repos").SetValue(strings.Join(repos, ","))

	return nil
}

func createAuth(ep transport.Endpoint) (transport.AuthMethod, error) {
	switch ep.Protocol() {
	case "ssh":
		auth, err := ssh.NewSSHAgentAuth(ep.User())
		if err != nil {
			return nil, err
		}
		return auth, nil
	default:
		return nil, nil
	}
}

func contains(arr []string, i string) bool {
	for _, e := range arr {
		if e == i {
			return true
		}
	}
	return false
}
