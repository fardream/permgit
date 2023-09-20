# permgit

[![Go Reference](https://pkg.go.dev/badge/github.com/fardream/permgit.svg)](https://pkg.go.dev/github.com/fardream/permgit)

## Commit, Tree, and Blob

Each commit in git contains the following

- tree hash
- parent hashes, lexicographical sorted.
- the author name and email, and the author timestamp.
- the committor name and email, and the committor timestamp.
- commmit message.

hash of the commit is the hash of the above content.

Tree is like a folder structure, all entries lexicographical sorted by names. A tree entry is

- file mode (regular file, executable file, symlink, tree)
- name
- hash

A file (either regular or executable), or blob contains only its content, and the hash is the hash of the header + content.

## Create New Linear History

Provided a linear history of commits, we can use the same author/committor/commit message and filter the tree in the commit and create a deterministic new commit.

## Example

```go
package main

import (
	"context"
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
func main() {
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
	hist, err := permgit.GetLinearHistory(context.Background(), headcommit, plumbing.ZeroHash, 10)
	orPanic(err)

	// select 3 files
	orfilter := permgit.NewOrFilter(
		permgit.NewPrefixFilter("/README.md"),
		permgit.NewPrefixFilter("/LICENSE"),
		permgit.NewPrefixFilter("/capis.go"),
	)

	// output storer
	outputfs := memory.NewStorage()

	newhist, err := permgit.FilterLinearHistory(context.Background(), hist, outputfs, orfilter)
	orPanic(err)

	// Note the result is deterministic
	fmt.Printf("From %d commits, generated %d commits.\nHead commit is:\n", len(hist), len(newhist))
	fmt.Println(newhist[5].String())
}
```
