package main

import (
	"context"
	"os/signal"
	"slices"
	"syscall"

	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/spf13/cobra"

	"github.com/fardream/permgit"
	"github.com/fardream/permgit/cmd"
)

func main() {
	newCmd().Execute()
}

type Cmd struct {
	*cobra.Command

	cmd.HistCmd
	dir string

	cmd.SetBranchCmd
	cmd.LogCmd
}

func newCmd() *Cmd {
	c := &Cmd{
		Command: &cobra.Command{
			Use:   "remove-git-gpg",
			Short: "remove gpg signature from series of commit.",
			Long:  "remove gpg signature from series of commit.",
			Args:  cobra.NoArgs,
		},
	}

	c.Run = c.run

	c.Flags().StringVarP(&c.dir, "dir", "i", c.dir, "input directory containing original git repo")
	c.MarkFlagRequired("dir")
	c.MarkFlagDirname("dir")

	c.Flags().IntVarP(&c.NumCommit, "num-commit", "n", c.NumCommit, "number of commits to seek back")
	c.Flags().StringVarP(&c.EndCommit, "end-commit", "e", c.EndCommit, "commit hash (default to head)")
	c.Flags().StringVarP(&c.StartCommit, "start-commit", "s", c.StartCommit, "commit hash to start from, default to empty, and history will seek to root unless restricted by number of commit")

	c.Flags().StringVar(&c.Branch, "branch", c.Branch, "branch to set the head to")
	c.Flags().BoolVar(&c.SetHead, "set-head", c.SetHead, "set the generated commit history as the head")

	c.Flags().IntVar(&c.LogLevel, "log-level", c.LogLevel, "log level passing to slog.")

	return c
}

func (c *Cmd) run(*cobra.Command, []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c.InitLog()

	chc := cache.NewObjectLRUDefault()

	inputfs := cmd.NewFileSystem(c.dir, chc)

	hist := c.GetHistory(ctx, inputfs)

	newhist := cmd.GetOrPanic(permgit.RemoveGPGForLinearHistory(ctx, hist, inputfs))

	if c.Branch != "" {
		slices.Reverse(newhist)
		for _, v := range newhist {
			if v != nil {
				c.SetBrancHead(inputfs, v.Hash)
				break
			}
		}
	} else if c.SetHead {
		cmd.Logger().Warn("empty branch name, head will not be set")
	}
}
