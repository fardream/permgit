// filter-git-hist is a more robust but limited git-filter-branch.
//
// The generated history is deterministic, and each run, as long as the parameters stay the same, will be exactly the same.
//
// The input commit history must be linear, there must not be submodules (they will be silently ignored), and
// GPG signature will also be dropped. The output blobs/trees/commits will be written to a different/output directory.
// Input/output are directly read/written from the .git folder of git repositories. For output, an empty .git is sufficient.
//
// The generated commit history can be set to a branch as defined by branch name parameter, and can also be optionally
// set as the head of the repo.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/cobra"

	"github.com/fardream/permgit"
	"github.com/fardream/permgit/cmd"
)

func main() {
	newCmd().Execute()
}

type Cmd struct {
	*cobra.Command

	inputdir  string
	outputdir string
	overwrite bool
	cmd.HistCmd

	cmd.SetBranchCmd
	cmd.LogCmd
	cmd.FilterCmd
}

const longDescription = `filter-git-hist is a more robust but limited git-filter-branch.

The generated history is deterministic, and each run, as long as the parameters stay the same, will be exactly the same.

The input commit history must be linear, there must not be submodules (they will be silently ignored), and
GPG signature will also be dropped. The output blobs/trees/commits will be written to a different/output directory.
Input/output are directly read/written from the .git folder of git repositories. For output, an empty .git is sufficient.

The generated commit history can be set to a branch as defined by branch name parameter, and can also be optionally
set as the head of the repo.
`

func newCmd() *Cmd {
	c := &Cmd{
		Command: &cobra.Command{
			Use:   "filter-git-hist",
			Short: "filter files and recreate git history",
			Long:  longDescription,
			Args:  cobra.NoArgs,
		},
	}

	c.Flags().StringArrayVarP(&c.Patterns, "pattern", "p", c.Patterns, "pattern to include in the generated history")
	c.MarkFlagRequired("pattern")
	c.Flags().StringVarP(&c.inputdir, "input-dir", "i", c.inputdir, "input directory containing original git repo")
	c.MarkFlagRequired("input-dir")
	c.MarkFlagDirname("input-dir")
	c.Flags().StringVarP(&c.outputdir, "output-dir", "o", c.outputdir, "output directory")
	c.MarkFlagRequired("output-dir")
	c.MarkFlagDirname("output-dir")
	c.Flags().BoolVarP(&c.overwrite, "overwrite", "w", c.overwrite, "overwrite the destination if it's already exists")
	c.Flags().IntVarP(&c.NumCommit, "num-commit", "n", c.NumCommit, "number of commits to seek back")
	c.Flags().StringVarP(&c.EndCommit, "end-commit", "e", c.EndCommit, "commit hash (default to head)")
	c.Flags().StringVarP(&c.StartCommit, "start-commit", "s", c.StartCommit, "commit hash to start from, default to empty, and history will seek to root unless restricted by number of commit")

	c.Flags().StringVar(&c.Branch, "branch", c.Branch, "branch to set the head to")
	c.Flags().BoolVar(&c.SetHead, "set-head", c.SetHead, "set the generated commit history as the head")

	c.Flags().IntVar(&c.LogLevel, "log-level", c.LogLevel, "log level passing to slog.")

	c.Run = c.run

	return c
}

func newOutputDir(outputdir string, overwrite bool, cache cache.Object) *filesystem.Storage {
	_, err := os.Stat(outputdir)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		cmd.OrPanic(os.MkdirAll(outputdir, 0o755))
	} else if err != nil {
		cmd.OrPanic(err)
	} else {
		entries := cmd.GetOrPanic(os.ReadDir(outputdir))
		if len(entries) != 0 && !overwrite {
			cmd.OrPanic(fmt.Errorf("directory %s is not empty; consider set overwrite", outputdir))
		}
	}

	return cmd.NewFileSystem(outputdir, cache)
}

func (c *Cmd) run(*cobra.Command, []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c.InitLog()
	chc := cache.NewObjectLRUDefault()

	inputfs := cmd.NewFileSystem(c.inputdir, chc)

	hist := c.GetHistory(ctx, inputfs)

	orfilter := c.GetFilter()
	outputfs := newOutputDir(c.outputdir, c.overwrite, chc)

	newhist := cmd.GetOrPanic(permgit.FilterLinearHistory(ctx, hist, outputfs, orfilter))

	c.SetBrancHeadFromHistory(inputfs, newhist)
}
