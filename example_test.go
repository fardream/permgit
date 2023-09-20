package permgit_test

import (
	"fmt"
	"log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/fardream/permgit"
)

func orPanic(err error) {
	if err != nil {
		log.Panic(err)
	}
}

// Example cloning a repo into in-memory store, select several commits from a specific commit, and filter it into another in-memory store.
func Example() {
	// URL for the repo
	url := "https://github.com/fardream/gmsk"
	// commit to start from
	headcommithash := plumbing.NewHash("e0235243feee0ec1bde865be5fa2c0b761eff804")

	// Clone repo
	r, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: url,
	})
	orPanic(err)

	// find the commit
	headcommit, err := r.CommitObject(headcommithash)
	orPanic(err)

	// obtain the history of the repo.
	hist, err := permgit.GetLinearHistory(headcommit, plumbing.ZeroHash, 10)
	orPanic(err)

	// select 3 files
	orfilter := permgit.NewOrFilter(
		permgit.NewPrefixFilter("/README.md"),
		permgit.NewPrefixFilter("/LICENSE"),
		permgit.NewPrefixFilter("/capis.go"),
	)

	// output storer
	outputfs := memory.NewStorage()

	newhist, err := permgit.FilterLinearHistory(hist, outputfs, orfilter)
	orPanic(err)

	// Note the result is deterministic
	fmt.Printf("From %d commits, generated %d commits.\nHead commit is:\n", len(hist), len(newhist))
	fmt.Println(newhist[5].String())

	// Output:
	// From 10 commits, generated 6 commits.
	// Head commit is:
	// commit 65e88d11b1331c3031945587c4c28635886fdc92
	// Author: Chao Xu <fardream@users.noreply.github.com>
	// Date:   Sat Sep 02 20:19:42 2023 -0400
	//
	//     Update doc for slice input. (#57)
}
