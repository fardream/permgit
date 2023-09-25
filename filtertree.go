package permgit

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// FilterTree filters the entries of the tree by the filter and stores it in the given [storer.Storer].
// If after filtering the tree is empty, nil will be returned for the tree and the error.
//
// Note: Submodules will be silently ignored.
func FilterTree(
	ctx context.Context,
	t *object.Tree,
	prepath []string,
	s storer.Storer,
	filter Filter,
) (*object.Tree, error) {
	newEntries := make([]object.TreeEntry, 0, len(t.Entries))

	for _, e := range t.Entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		fullname := addpath(prepath, e.Name)
		fullnamestring := pathsToFullPath(fullname)

		switch e.Mode {
		case filemode.Deprecated, filemode.Executable, filemode.Regular, filemode.Symlink:
			if !filter.Filter(fullname, false).IsIn() {
				continue
			}
			entryToAdd := e
			file, err := t.TreeEntryFile(&entryToAdd)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to obtain path %s: %w",
					fullnamestring,
					err)
			}

			haserr := s.HasEncodedObject(file.Hash)
			if haserr != nil {
				if err := updateHashAndSave(ctx, file, s); err != nil {
					return nil, errorf(
						err,
						"failed to write %s %s into new repo: %w",
						e.Mode.String(),
						fullnamestring,
						err)
				}
			}
			newEntries = append(newEntries, entryToAdd)
		case filemode.Submodule:
			logger.Warn("ignoring submodule", "path", fullnamestring)
			continue
		case filemode.Empty:
			continue
		case filemode.Dir:
			dir, err := t.Tree(e.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to find sub tree %s: %w", fullnamestring, err)
			}
			var newTree *object.Tree
			switch filter.Filter(fullname, true) {
			case FilterResult_Out:
				continue
			case FilterResult_In:
				if err = CopyTree(ctx, dir, s); err != nil {
					return nil, errorf(err, "failed to copy sub tree %s: %w", fullnamestring, err)
				}

				newTree, err = object.GetTree(s, dir.Hash)
				if err != nil {
					return nil, fmt.Errorf("failed to get tree %s: %w", fullnamestring, err)
				}
			case FilterResult_DirDive:
				newTree, err = FilterTree(ctx, dir, fullname, s, filter)
				if err != nil {
					return nil, err
				}
			}
			if newTree == nil {
				continue
			}

			newEntries = append(newEntries, object.TreeEntry{
				Name: e.Name,
				Mode: e.Mode,
				Hash: newTree.Hash,
			})
		}
	}

	if len(newEntries) == 0 {
		logger.Debug("empty tree", "tree", t.Hash, "prefix", pathsToFullPath(prepath))
		return nil, nil
	}

	newTree := &object.Tree{
		Entries: newEntries,
	}

	newHash, err := GetHash(newTree)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash for new tree %s: %w", pathsToFullPath(prepath), err)
	}

	newTree.Hash = *newHash

	if err := updateHashAndSave(ctx, newTree, s); err != nil {
		return nil, errorf(err, "failed to save the new tree %s: %w", pathsToFullPath(prepath), err)
	}

	return newTree, nil
}
