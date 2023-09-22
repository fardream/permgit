package permgit

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// ExpandCommit added the changes contained in the filteredNew to filteredOrig and try to apply them to target, it will generate a new commit.
func ExpandCommit(
	ctx context.Context,
	sourceStorer storer.Storer,
	filteredOrig *object.Commit,
	filteredNew *object.Commit,
	target *object.Commit,
	targetStorer storer.Storer,
	filter TreeEntryFilter,
) (*object.Commit, error) {
	newtarget := &object.Commit{
		Committer:    filteredNew.Committer,
		Author:       filteredNew.Author,
		Message:      filteredNew.Message,
		ParentHashes: []plumbing.Hash{target.Hash},
	}

	filteredOrigTree, err := filteredOrig.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain filtered parent tree: %w", err)
	}
	filteredNewTree, err := filteredNew.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain filtered new tree: %w", err)
	}
	targetOrigTree, err := target.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain target parent tree: %w", err)
	}

	newtree, err := ExpandTree(ctx, sourceStorer, filteredOrigTree, filteredNewTree, targetOrigTree, targetStorer, filter)
	if err != nil {
		return nil, errorf(err, "failed to expand tree for target: %w", err)
	}
	if newtree != nil {
		newtarget.TreeHash = newtree.Hash
	} else {
		logger.Warn("empty tree", "filtered-new-commit", filteredNew.Hash, "filtered-orig-commit", filteredOrig.Hash, "target", target.Hash)
	}

	err = updateHashAndSave(ctx, newtarget, targetStorer)
	if err != nil {
		return nil, errorf(err, "failed to update new tree into storage: %w", err)
	}

	return newtarget, nil
}
