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

See [Example](https://pkg.go.dev/github.com/fardream/permgit#example-package)
