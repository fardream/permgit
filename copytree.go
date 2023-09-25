package permgit

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// CopyTree copies the given tree into the [storer.Storer].
// If the tree already exists in s, function returns nil error right away.
func CopyTree(ctx context.Context, t *object.Tree, s storer.Storer) error {
	if s.HasEncodedObject(t.Hash) == nil {
		logger.Debug("tree exists, not copying", "hash", t.Hash)
		return nil
	}

	logger.Debug("copy tree", "hash", t.Hash)
	for _, e := range t.Entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		switch e.Mode {
		case filemode.Deprecated, filemode.Executable, filemode.Regular, filemode.Symlink:
			if s.HasEncodedObject(e.Hash) == nil {
				continue
			}
			file, err := t.TreeEntryFile(&e)
			if err != nil {
				return fmt.Errorf("failed to obtain file %s: %w", e.Hash, err)
			}

			if err := updateHashAndSave(ctx, file, s); err != nil {
				return errorf(err, "failed to write %s %s into new repo: %w", e.Mode.String(), file.Hash, err)
			}
		case filemode.Submodule:
			logger.Warn("ignoring submodule", "path", e.Name)
		case filemode.Empty:
			continue
		case filemode.Dir:
			dir, err := t.Tree(e.Name)
			if err != nil {
				return fmt.Errorf("failed to find sub tree %s %s: %w", e.Name, e.Hash, err)
			}

			if err := CopyTree(ctx, dir, s); err != nil {
				return errorf(err, "failed to copy sub tree %s %s: %w", e.Name, e.Hash, err)
			}
		}
	}

	newtree := object.Tree{
		Hash:    t.Hash,
		Entries: t.Entries,
	}

	if err := updateHashAndSave(ctx, &newtree, s); err != nil {
		return errorf(err, "failed to save tree %s: %w", newtree.Hash, err)
	}

	return nil
}
