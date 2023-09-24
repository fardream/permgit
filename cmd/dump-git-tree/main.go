// dump-git-tree dumps the git tree and optionally apply pattern filters.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-git/go-git/v5/plumbing"
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

	dir    string
	commit string
	tree   string
	branch string
	head   bool

	cmd.FilterCmd

	outfilename string

	cmd.LogCmd
}

func newCmd() *Cmd {
	c := &Cmd{
		Command: &cobra.Command{
			Use:   "dump-git-tree",
			Short: "dump the git tree and optionally apply pattern filters.",
			Long:  "dump-git-tree dump the git tree and optionally apply pattern filters.\n" + cmd.PatternDescription,
			Args:  cobra.NoArgs,
		},
	}

	c.Run = c.run

	c.SetupFilterCobra(c.Command, false)
	c.Flags().StringVarP(&c.dir, "dir", "i", c.dir, "input directory containing original git repo")
	c.MarkFlagRequired("dir")
	c.MarkFlagDirname("dir")

	c.Flags().StringVarP(&c.commit, "commit", "c", c.commit, "commit")
	c.Flags().StringVarP(&c.tree, "tree", "t", c.tree, "tree")
	c.Flags().StringVarP(&c.branch, "branch", "b", c.branch, "branch")
	c.Flags().BoolVar(&c.head, "head", c.head, "use head")
	c.MarkFlagsMutuallyExclusive("commit", "tree", "branch", "head")

	c.Flags().StringVarP(&c.outfilename, "output", "o", c.outfilename, "output file name, use - or leave empty for stdout")
	c.MarkFlagFilename("output")

	c.Flags().IntVar(&c.LogLevel, "log-level", c.LogLevel, "log level passing to slog.")

	return c
}

func (c *Cmd) run(*cobra.Command, []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	c.InitLog()

	chc := cache.NewObjectLRUDefault()

	fs := cmd.NewFileSystem(c.dir, chc)

	var hash plumbing.Hash
	switch {
	case c.branch != "":
		branch := cmd.GetOrPanic(fs.Reference(plumbing.NewBranchReferenceName(c.branch)))
		if branch.Hash().IsZero() {
			branch = cmd.GetOrPanic(fs.Reference(branch.Target()))
		}
		hash = cmd.GetOrPanic(object.GetCommit(fs, branch.Hash())).TreeHash
	case c.commit != "":
		hash = cmd.GetOrPanic(object.GetCommit(fs, cmd.MustHash(c.commit))).TreeHash
	case c.head:
		head := cmd.GetOrPanic(fs.Reference(plumbing.HEAD))
		if head.Hash().IsZero() {
			head = cmd.GetOrPanic(fs.Reference(head.Target()))
		}
		hash = cmd.GetOrPanic(object.GetCommit(fs, head.Hash())).TreeHash
	case c.tree != "":
		hash = cmd.MustHash(c.tree)
	default:
		cmd.OrPanic(fmt.Errorf("require one of branch, head, tree, or commit"))
	}

	tree := cmd.GetOrPanic(object.GetTree(fs, hash))

	filter := c.GetFilter()

	var out io.WriteCloser
	if c.outfilename == "" || c.outfilename == "-" {
		out = os.Stdout
	} else {
		out = cmd.GetOrPanic(os.Create(c.outfilename))
		defer out.Close()
	}

	permgit.DumpTree(ctx, nil, tree, filter, out)
}
