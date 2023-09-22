package permgit

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
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
	ctx context.Context,
	c *object.Commit,
	parent *object.Commit,
	s storer.Storer,
	filters TreeEntryFilter,
) (*object.Commit, error) {
	t, err := c.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain tree for commit %s: %w", c.Hash.String(), err)
	}

	newtree, err := FilterTree(ctx, t, "", s, filters)
	if err != nil {
		return nil, errorf(err, "failed to filter tree: %w", err)
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

	newhash, err := GetHash(newcommit)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain new hash for commit: %w ", err)
	}

	newcommit.Hash = *newhash

	if err := updateHashAndSave(ctx, newcommit, s); err != nil {
		return nil, errorf(err, "failed to save commit: %w", err)
	}

	return newcommit, nil
}
