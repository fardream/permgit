package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/cobra"

	"github.com/fardream/permgit"
)

func main() {
	newCmd().Execute()
}

func orPanic(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getOrPanic[T any](a T, err error) T {
	orPanic(err)

	return a
}

type Cmd struct {
	*cobra.Command

	prefixes    []string
	inputdir    string
	outputdir   string
	overwrite   bool
	numCommit   int
	endCommit   string
	startCommit string

	branch  string
	sethead bool

	loglevel int
}

const longDescription = `filter-git-hist is a more robust but limited git-filter-branch.

The generated history is deterministic, and each run, as long as the parameters stay the same, will be exactly the same.

The input commit history must be linear, there must not be submodules (they will be silently ignored), and
GPG signature will also be dropped. The output blobs/trees/commits will be written to a different/output directory.
Input/output are directly read/written from the .git folder of git repositories. For output, an empty .git is sufficient.

The generated commit history can be set to a branch as defined by branch name parameter, and can also be optionally
set as the head of the repo.
`

func newCmd() (cmd *Cmd) {
	cmd = &Cmd{
		Command: &cobra.Command{
			Use:   "filter-git-hist",
			Short: "filter files and recreate git history",
			Long:  longDescription,
			Args:  cobra.NoArgs,
		},
	}

	cmd.Flags().StringArrayVarP(&cmd.prefixes, "prefix", "p", cmd.prefixes, "prefixes to include in the generated history")
	cmd.MarkFlagRequired("prefix")
	cmd.Flags().StringVarP(&cmd.inputdir, "input-dir", "i", cmd.inputdir, "input directory containing original git repo")
	cmd.MarkFlagRequired("input-dir")
	cmd.MarkFlagDirname("input-dir")
	cmd.Flags().StringVarP(&cmd.outputdir, "output-dir", "o", cmd.outputdir, "output directory")
	cmd.MarkFlagRequired("output-dir")
	cmd.MarkFlagDirname("output-dir")
	cmd.Flags().BoolVarP(&cmd.overwrite, "overwrite", "w", cmd.overwrite, "overwrite the destination if it's already exists")
	cmd.Flags().IntVarP(&cmd.numCommit, "num-commit", "n", cmd.numCommit, "number of commits to seek back")
	cmd.Flags().StringVarP(&cmd.endCommit, "end-commit", "e", cmd.endCommit, "commit hash (default to head)")
	cmd.Flags().StringVarP(&cmd.startCommit, "start-commit", "s", cmd.startCommit, "commit hash to start from, default to empty, and history will seek to root unless restricted by number of commit")

	cmd.Flags().StringVar(&cmd.branch, "branch", cmd.branch, "branch to set the head to")
	cmd.Flags().BoolVar(&cmd.sethead, "set-head", cmd.sethead, "set the generated commit history as the head")

	cmd.Flags().IntVar(&cmd.loglevel, "log-level", cmd.loglevel, "log level passing to slog.")

	cmd.Run = cmd.run

	return
}

func newOutputDir(outputdir string, overwrite bool, cache cache.Object) *filesystem.Storage {
	_, err := os.Stat(outputdir)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		orPanic(os.MkdirAll(outputdir, 0o755))
	} else if err != nil {
		orPanic(err)
	} else {
		entries := getOrPanic(os.ReadDir(outputdir))
		if len(entries) != 0 && !overwrite {
			orPanic(fmt.Errorf("directory %s is not empty; consider set overwrite", outputdir))
		}
	}

	return filesystem.NewStorage(osfs.New(getOrPanic(filepath.Abs(outputdir))), cache)
}

func (cmd *Cmd) run(*cobra.Command, []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	loglevel := new(slog.LevelVar)
	loglevel.Set(slog.Level(cmd.loglevel))
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: loglevel})))

	absinput := getOrPanic(filepath.Abs(cmd.inputdir))

	slog.Debug("remap input dir", "input", cmd.inputdir, "absinput", absinput)

	inputbasefs := osfs.New(absinput)
	chc := cache.NewObjectLRUDefault()
	inputfs := filesystem.NewStorage(inputbasefs, chc)

	head := getOrPanic(inputfs.Reference(plumbing.HEAD))

	if head.Hash().IsZero() {
		head = getOrPanic(inputfs.Reference(head.Target()))
	}
	endHash := head.Hash()

	var startHash plumbing.Hash
	if cmd.startCommit != "" {
		startHash = plumbing.NewHash(cmd.startCommit)
	}

	if cmd.endCommit != "" {
		endHash = plumbing.NewHash(cmd.endCommit)
	}

	slog.Debug("head hash", "head", endHash)

	c := getOrPanic(inputfs.EncodedObject(plumbing.CommitObject, endHash))

	cmt := getOrPanic(object.DecodeCommit(inputfs, c))

	orfilter := permgit.NewOrFilter()
	for _, prefix := range cmd.prefixes {
		orfilter.Add(permgit.NewPrefixFilter(prefix))
	}

	hist := getOrPanic(permgit.GetLinearHistory(ctx, cmt, startHash, cmd.numCommit))

	outputfs := newOutputDir(cmd.outputdir, cmd.overwrite, chc)

	newhist := getOrPanic(permgit.FilterLinearHistory(ctx, hist, outputfs, orfilter))

	if cmd.branch != "" {
		slices.Reverse(newhist)
		for _, v := range newhist {
			if v != nil {
				refname := plumbing.NewBranchReferenceName(cmd.branch)
				ref := plumbing.NewHashReference(refname, v.Hash)
				orPanic(outputfs.SetReference(ref))
				if cmd.sethead {
					headref := plumbing.NewSymbolicReference(plumbing.HEAD, refname)
					orPanic(outputfs.SetReference(headref))
				}
				break
			}
		}
	} else {
		if cmd.sethead {
			slog.Warn("empty branch name, head will not be set")
		}
	}
}
