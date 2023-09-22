package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"

	"github.com/fardream/permgit"
	"github.com/fardream/permgit/cmd"
)

func main() {
	newCmd().Execute()
}

type Cmd struct {
	*cobra.Command

	prefixes  []string
	inputdir  string
	outputdir string

	inputCommit  string
	targetCommit string

	cmd.SetBranchCmd

	cmd.LogCmd
}

const longDescription = `expand-git-commit adds back the changes made in a repo filtered by filter-git-hist to the unfiltered repo.

The target commit, once filtered down by input filters, should generate exact same tree as the input commit's parent.
The generated commit is deterministic, and each run, as long as the parameters stay the same, will be exactly the same.

The process will panic if any of the files in change set is filtered out by the input filters.

The input/output directory are .git repositories.

The generated commit can be set to a branch as defined by the branch name, and can also be optionally set as the head of the repo.
`

func newCmd() *Cmd {
	c := &Cmd{
		Command: &cobra.Command{
			Use:   "expand-git-commit",
			Short: "add changes in filtered commit back to unfiltered commit.",
			Long:  longDescription,
			Args:  cobra.NoArgs,
		},
	}

	c.Flags().StringArrayVarP(&c.prefixes, "prefix", "p", c.prefixes, "prefixes use to filter repo")
	c.MarkFlagRequired("prefix")
	c.Flags().StringVarP(&c.inputdir, "input-dir", "i", c.inputdir, "input directory containing filtered git repo")
	c.MarkFlagRequired("input-dir")
	c.MarkFlagDirname("input-dir")
	c.Flags().StringVarP(&c.outputdir, "output-dir", "o", c.outputdir, "output directory, containing the unfiltered repo.")
	c.MarkFlagRequired("output-dir")
	c.MarkFlagDirname("output-dir")
	c.Flags().StringVarP(&c.inputCommit, "input-commit", "c", c.inputCommit, "input commit, which is in the filtered/input repo.")
	c.MarkFlagRequired("input-commit")
	c.Flags().StringVarP(&c.targetCommit, "target-commit", "t", c.targetCommit, "target commit, changes will be applied to this commit and a new commit created.")
	c.MarkFlagRequired("target-commit")

	c.Flags().StringVar(&c.Branch, "branch", c.Branch, "branch to set the head to")
	c.Flags().BoolVar(&c.SetHead, "set-head", c.SetHead, "set the generated commit history as the head")

	c.Flags().IntVar(&c.LogLevel, "log-level", c.LogLevel, "log level passing to slog.")

	c.Run = c.run

	return c
}

func (c *Cmd) run(*cobra.Command, []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c.InitLog()

	chc := cache.NewObjectLRUDefault()

	inputfs := cmd.NewFileSystem(c.inputdir, chc)

	outputfs := cmd.NewFileSystem(c.outputdir, chc)

	inputcommit := cmd.GetOrPanic(object.GetCommit(inputfs, cmd.MustHash(c.inputCommit)))
	targetcommit := cmd.GetOrPanic(object.GetCommit(outputfs, cmd.MustHash(c.targetCommit)))
	inputparent := cmd.GetOrPanic(inputcommit.Parent(0))

	filter := permgit.NewOrFilterForPrefixes(c.prefixes...)

	newcommit := cmd.GetOrPanic(permgit.ExpandCommit(
		ctx,
		inputfs,
		inputparent,
		inputcommit,
		targetcommit,
		outputfs,
		filter,
	))

	cmd.Logger().Debug("newcommit", "hash", newcommit.Hash)

	if c.Branch != "" {
		c.SetBrancHead(outputfs, newcommit.Hash)
	} else if c.SetHead {
		cmd.Logger().Warn("empty branch name, head will not be set")
	}
}
