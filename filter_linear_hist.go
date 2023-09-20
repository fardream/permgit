package permgit

import (
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// FilterLinearHistory performs filters on a sequence of commits of a linear history and
// produces new commits in the provided [storer.Store].
// The first commit is the earliest commit, and the last one is the latest or head.
//
//   - The first commit will become the new root of the filtered repo.
//   - Filtered commits containing empty trees cause all previous commits to be dropped.
//     The next commit with non-empty tree will become the new root.
//   - Filtered commits containing the exact same tree as its parent will also be dropped,
//     and commit after it will consider its parent its own parent.
//
// The newly created commits will have exact same author info, committor info, commit message,
// but will parent correctly linked and gpg sign information dropped.
//
// The input commits can be obtained from [GetLinearHistory].
func FilterLinearHistory(
	hist []*object.Commit,
	s storer.Storer,
	filter TreeEntryFilter,
) ([]*object.Commit, error) {
	newhist := make([]*object.Commit, 0, len(hist))

	var prevCommit *object.Commit

	for i, v := range hist {
		newcommit, err := FilterCommit(v, prevCommit, s, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to generate commit at %d for commit %s: %w ", i, v.Hash, err)
		}

		commitinfo := "empty"
		if newcommit != nil {
			commitinfo = fmt.Sprintf("%s by %s <%s>", newcommit.Hash, newcommit.Author.Name, newcommit.Author.Email)
		} else {
			newhist = newhist[:0]
		}

		slog.Info("processing commit", "id", i, "hash", v.Hash, "newcommit", commitinfo)

		if newcommit != prevCommit && newcommit != nil {
			newhist = append(newhist, newcommit)
		}

		prevCommit = newcommit
	}

	return newhist, nil
}
