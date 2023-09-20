package permgit

import (
	"fmt"
	"math"
	"slices"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetLinearHistory produces the linear history from a given head commit.
//   - the number of commits can be limit by number of commits included in the history.
//     A limit <= 0 indicates no limit on how many commits can be returned
//   - the start commit can be specified by the startHash
//
// It returns an error when more than one parents exist for the commit in the historical list.
func GetLinearHistory(
	head *object.Commit,
	startHash plumbing.Hash,
	numCommit int,
) ([]*object.Commit, error) {
	result := make([]*object.Commit, 0)

	if numCommit <= 0 {
		numCommit = math.MaxInt
	}

	current := head
	nseen := 1
	for {
		result = append(result, current)
		if startHash == current.Hash {
			break
		}
		nseen += 1
		if nseen > numCommit {
			break
		}
		numparent := current.NumParents()
		if numparent > 1 {
			return nil, fmt.Errorf("commit %s has %d parents, and not linear.", current.Hash.String(), numparent)
		} else if numparent == 0 {
			break
		}
		var err error
		current, err = current.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain parent for commit %s: %w", current.Hash.String(), err)
		}
	}

	slices.Reverse(result)

	return result, nil
}
