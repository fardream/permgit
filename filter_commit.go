package permgit

import (
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// FilterCommit creates a new [object.Commit] in the given [storer.Storer]
// by applying filters to the tree in the input [object.Commit].
// Optionally a parent commit can set on the generated commit.
// The author info, committor info, commit message will be copied from the input commit.
// Howver, GPG sign information will be dropped.
//
//   - If after filtering, the tree is empty, a nil will be returned, and error will also be nil.
//   - If the generated tree is exactly the same as the parent's, the parent commit will be returned and no new commit will be generated.
//
// Submodules will be silently ignored.
func FilterCommit(
	c *object.Commit,
	parent *object.Commit,
	s storer.Storer,
	filters TreeEntryFilter,
) (*object.Commit, error) {
	t, err := c.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain tree for commit %s: %w", c.Hash.String(), err)
	}

	newtree, err := FilterTree(t, "", s, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to filter tree: %w", err)
	}

	if newtree == nil {
		return nil, nil
	}

	var parents []plumbing.Hash
	if parent != nil {
		if parent.TreeHash == newtree.Hash {
			return parent, nil
		}
		parents = append(parents, parent.Hash)
	}

	newcommit := &object.Commit{
		TreeHash:     newtree.Hash,
		Author:       c.Author,
		Committer:    c.Committer,
		Message:      c.Message,
		ParentHashes: parents,
	}

	if err := updateHashAndSave(newcommit, s); err != nil {
		return nil, fmt.Errorf("failed to save commit: %w", err)
	}

	return newcommit, nil
}

// FilterTree filters the entries of the tree by the filter and stores it in the given [storer.Storer].
// If after filtering the tree is empty, nil will be returned for the tree and the error.
//
// Note: Submodules will be silently ignored.
func FilterTree(t *object.Tree, prepath string, s storer.Storer, filter TreeEntryFilter) (*object.Tree, error) {
	newEntries := make([]object.TreeEntry, 0, len(t.Entries))
	for _, e := range t.Entries {
		switch e.Mode {
		case filemode.Deprecated, filemode.Executable, filemode.Regular, filemode.Symlink:
			fullname := prepath + "/" + e.Name
			if !filter.IsIn(fullname) {
				continue
			}
			entryToAdd := e
			file, err := t.TreeEntryFile(&entryToAdd)
			if err != nil {
				return nil, fmt.Errorf("failed to obtain path %s: %w", fullname, err)
			}

			haserr := s.HasEncodedObject(file.Hash)
			if haserr != nil {
				if err := updateHashAndSave(file, s); err != nil {
					return nil, fmt.Errorf("failed to write %s %s into new repo: %w", e.Mode.String(), fullname, err)
				}
			}
			newEntries = append(newEntries, entryToAdd)
		case filemode.Submodule:
			slog.Warn("ignoring submodule", "path", prepath+"/"+e.Name)
			continue
		case filemode.Empty:
			continue
		case filemode.Dir:
			fullname := prepath + "/" + e.Name
			dir, err := t.Tree(e.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to find sub tree %s: %w", fullname, err)
			}
			newTree, err := FilterTree(dir, fullname, s, filter)
			if err != nil {
				return nil, err
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
		slog.Debug("empty tree", "tree", t.Hash)
		return nil, nil
	}

	newTree := &object.Tree{
		Entries: newEntries,
	}

	if err := updateHashAndSave(newTree, s); err != nil {
		return nil, fmt.Errorf("failed to save the new tree %s: %w", prepath, err)
	}

	return newTree, nil
}
