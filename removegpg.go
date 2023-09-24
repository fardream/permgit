package permgit

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

func RemoveGPGForLinearHistory(ctx context.Context, hist []*object.Commit, s storer.Storer) ([]*object.Commit, error) {
	newhist := make([]*object.Commit, 0, len(hist))

	var prevcommit *object.Commit

	for i, v := range hist {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var parenthashses []plumbing.Hash
		if prevcommit == nil {
			parenthashses = append(parenthashses, v.ParentHashes[0])
		} else {
			parenthashses = append(parenthashses, prevcommit.Hash)
		}
		newcommit := &object.Commit{
			Author:       v.Author,
			Committer:    v.Committer,
			Message:      v.Message,
			TreeHash:     v.TreeHash,
			ParentHashes: parenthashses,
		}

		newcommithash, err := GetHash(newcommit)
		if err != nil {
			return nil, fmt.Errorf("failed to get hash for new commit: %w", err)
		}
		newcommit.Hash = *newcommithash
		logger.Debug("remove gpgp", "id", i, "commit", v.Hash, "newcommit", newcommit.Hash)
		if err := updateHashAndSave(ctx, newcommit, s); err != nil {
			return nil, fmt.Errorf("failed to save new commit %s to storage: %w", newcommit.Hash.String(), err)
		}

		newhist = append(newhist, newcommit)
		prevcommit = newcommit
	}

	return newhist, nil
}
