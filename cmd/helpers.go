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

	"github.com/fardream/permgit"
)

var logger *slog.Logger = slog.Default()

func Logger() *slog.Logger {
	return logger
}

func OrPanic(err error) {
	if err != nil {
		logger.Error("error", "err", err)
		os.Exit(1)
	}
}

func GetOrPanic[T any](a T, err error) T {
	OrPanic(err)

	return a
}

func InitLogger(level int) {
	loglevel := new(slog.LevelVar)
	loglevel.Set(slog.Level(level))
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: loglevel}))
	permgit.SetLogger(logger)
}

func NewFileSystem(dir string, cache cache.Object) *filesystem.Storage {
	absdir := GetOrPanic(filepath.Abs(dir))

	if absdir != dir {
		logger.Debug("remap dir", "from", dir, "to", absdir)
	}

	return filesystem.NewStorage(osfs.New(absdir), cache)
}

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

type HistCmd struct {
	NumCommit   int
	EndCommit   string
	StartCommit string
}

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

type SetBranchCmd struct {
	Branch  string
	SetHead bool
}

func (c *SetBranchCmd) SetBrancHead(s storer.Storer, h plumbing.Hash) {
	refname := plumbing.NewBranchReferenceName(c.Branch)
	ref := plumbing.NewHashReference(refname, h)
	OrPanic(s.SetReference(ref))
	if c.SetHead {
		headref := plumbing.NewSymbolicReference(plumbing.HEAD, refname)
		OrPanic(s.SetReference(headref))
	}
}

type LogCmd struct {
	LogLevel int
}

func (c *LogCmd) InitLog() {
	InitLogger(c.LogLevel)
}
