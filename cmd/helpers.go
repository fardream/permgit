// cmd package contains helper functions for various commands.
package cmd

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/cobra"

	"github.com/fardream/permgit"
)

var logger *slog.Logger = slog.Default()

// Return the logger for the command
func Logger() *slog.Logger {
	return logger
}

// OrPanic panics if err is not nil
func OrPanic(err error) {
	if err != nil {
		logger.Error("error", "err", err)
		os.Exit(1)
	}
}

// GetOrPanic checks if err is nil, panics if not, otherwise return a
func GetOrPanic[T any](a T, err error) T {
	OrPanic(err)

	return a
}

// initLogger creates a new [slog.TextHandler] with the given level.
func initLogger(level int) {
	loglevel := new(slog.LevelVar)
	loglevel.Set(slog.Level(level))
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: loglevel}))
	permgit.SetLogger(logger)
}

// NewFileSystem obtains the absolute path to the dir and creates a new [filesystem.Storage]
func NewFileSystem(dir string, cache cache.Object) *filesystem.Storage {
	absdir := GetOrPanic(filepath.Abs(dir))

	if absdir != dir {
		logger.Debug("remap dir", "from", dir, "to", absdir)
	}

	return filesystem.NewStorage(osfs.New(absdir), cache)
}

// MustHash gets the 20 byte hash from string
func MustHash(s string) plumbing.Hash {
	if len(s) != 40 {
		OrPanic(fmt.Errorf("hex for hash %s doesn't have length 40", s))
	}

	b := GetOrPanic(hex.DecodeString(s))

	var r plumbing.Hash

	if copy(r[:], b) != 20 {
		OrPanic(fmt.Errorf("copied byte count is not 20"))
	}

	return r
}

// HistCmd are command components used to get history
type HistCmd struct {
	NumCommit   int
	EndCommit   string
	StartCommit string
}

// GetHistory returns the linear history
func (c *HistCmd) GetHistory(ctx context.Context, s storer.Storer) []*object.Commit {
	head := GetOrPanic(s.Reference(plumbing.HEAD))

	if head.Hash().IsZero() {
		head = GetOrPanic(s.Reference(head.Target()))
	}
	endHash := head.Hash()

	var startHash plumbing.Hash
	if c.StartCommit != "" {
		startHash = plumbing.NewHash(c.StartCommit)
	}

	if c.EndCommit != "" {
		endHash = plumbing.NewHash(c.EndCommit)
	}

	Logger().Debug("head hash", "head", endHash)

	headcommit := GetOrPanic(object.GetCommit(s, endHash))

	return GetOrPanic(permgit.GetLinearHistory(ctx, headcommit, startHash, c.NumCommit))
}

// SetBranchCmd is for output the commit to a branch and potentially set it to head.
type SetBranchCmd struct {
	Branch  string
	SetHead bool
}

// SetBrancHead sets the has as the branch
func (c *SetBranchCmd) SetBrancHead(s storer.Storer, h plumbing.Hash) {
	if c.Branch != "" {
		refname := plumbing.NewBranchReferenceName(c.Branch)
		ref := plumbing.NewHashReference(refname, h)
		OrPanic(s.SetReference(ref))
		if c.SetHead {
			headref := plumbing.NewSymbolicReference(plumbing.HEAD, refname)
			OrPanic(s.SetReference(headref))
		}
	} else if c.SetHead {
		Logger().Warn("empty branch name, head will not be set")
	}
}

func (c *SetBranchCmd) SetBrancHeadFromHistory(s storer.Storer, newhist []*object.Commit) {
	if c.Branch != "" {
		n := len(newhist)
		for i := 0; i < n; i++ {
			v := newhist[n-i-1]
			if v != nil {
				h := v.Hash
				refname := plumbing.NewBranchReferenceName(c.Branch)
				ref := plumbing.NewHashReference(refname, h)
				OrPanic(s.SetReference(ref))
				if c.SetHead {
					headref := plumbing.NewSymbolicReference(plumbing.HEAD, refname)
					OrPanic(s.SetReference(headref))
				}
			}
		}
	} else if c.SetHead {
		Logger().Warn("empty branch name, head will not be set")
	}
}

// LogCmd contains cmd's log configuration.
type LogCmd struct {
	LogLevel int
}

func (c *LogCmd) InitLog() {
	initLogger(c.LogLevel)
}

type FilterCmd struct {
	Patterns          []string
	PatternFile       string
	IgnoreUnsupported bool

	IsRequired bool
}

func (c *FilterCmd) SetupFilterCobra(cmd *cobra.Command, required bool) {
	cmd.Flags().StringArrayVarP(&c.Patterns, "pattern", "p", c.Patterns, "patterns use to filter repo")
	cmd.Flags().StringVar(&c.PatternFile, "pattern-file", c.PatternFile, "a .gitignore like file for patterns")
	cmd.MarkFlagFilename("pattern-file")
	cmd.Flags().BoolVar(&c.IgnoreUnsupported, "allow-unsupported-pattern", c.IgnoreUnsupported, "allow the parser to ignore patterns supported like !")
	if required {
		cmd.MarkFlagsOneRequired("pattern-file", "pattern")
		c.IsRequired = true
	}
}

func (c *FilterCmd) GetFilter() permgit.Filter {
	filelines := c.Patterns[:]
	if c.PatternFile != "" {
		content := GetOrPanic(os.ReadFile(c.PatternFile))
		filelines = append(filelines, GetOrPanic(permgit.LoadPatternStringFromString(string(content), c.IgnoreUnsupported))...)
	}

	if !c.IsRequired && len(filelines) == 0 {
		return &permgit.TrueFilter{}
	}

	return GetOrPanic(permgit.NewOrFilterForPatterns(filelines...))
}

const PatternDescription = `supported patterns for filtering:

- all patterns are or-ed - if a file is included by one of the patterns, it will be included.
- '**' is for multi level directories, and it can only appear once in the match.
- '*' is for match one level of names.
- '!' and escapes are unsupported.
- '#' and blank lines are ignored.
`
