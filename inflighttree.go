package permgit

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// inflightTree is a structure built from a [object.Tree], and records if the content of the tree should be changed.
// Use [inflightTree.BuildTree] to update the contained tree.
type inflightTree struct {
	nonTrees map[string]object.TreeEntry
	trees    map[string]*inflightTree

	changed bool

	baseTree *object.Tree
}

func newInflightTree(t *object.Tree) (*inflightTree, error) {
	r := &inflightTree{
		nonTrees: make(map[string]object.TreeEntry),
		trees:    make(map[string]*inflightTree),
		baseTree: t,
	}

	for _, e := range t.Entries {
		switch e.Mode {
		case filemode.Dir:
			subtree, err := t.Tree(e.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to obtain sub tree at %s: %w", e.Name, err)
			}
			subinflight, err := newInflightTree(subtree)
			if err != nil {
				return nil, fmt.Errorf("failed to create sub inflight tree at %s: %w", e.Name, err)
			}
			r.trees[e.Name] = subinflight
		default:
			r.nonTrees[e.Name] = e
		}
	}

	return r, nil
}

// BuildTree updates tree's contents and return the rebuilt tree.
func (it *inflightTree) BuildTree(ctx context.Context, s storer.Storer) (*object.Tree, error) {
	if !it.changed {
		return it.baseTree, nil
	}

	it.changed = false

	newtree := &object.Tree{}

	for name, subtree := range it.trees {
		_, err := subtree.BuildTree(ctx, s)
		if err != nil {
			return nil, errorf(err, "failed to build sub tree %s: %w", name, err)
		}

		newtree.Entries = append(newtree.Entries, object.TreeEntry{
			Name: name,
			Mode: filemode.Dir,
			Hash: subtree.baseTree.Hash,
		})
	}

	for _, item := range it.nonTrees {
		newtree.Entries = append(newtree.Entries, item)
	}

	slices.SortFunc(newtree.Entries, func(l, r object.TreeEntry) int {
		if l.Name < r.Name {
			return -1
		} else if l.Name > r.Name {
			return 1
		} else {
			return 0
		}
	})

	treehash, err := GetHash(newtree)
	if err != nil {
		return nil, fmt.Errorf("failed to get hash: %w", err)
	}
	newtree.Hash = *treehash

	if err := updateHashAndSave(ctx, newtree, s); err != nil {
		return nil, errorf(err, "failed to save new tree: %w", err)
	}

	newtree, err = object.GetTree(s, newtree.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to reobtain the tree: %w", err)
	}

	logger.Debug("updating tree", "old", treeEntryNames(it.baseTree.Entries), "new", treeEntryNames(newtree.Entries))
	it.baseTree = newtree

	return it.baseTree, nil
}

func treeEntryNames(t []object.TreeEntry) []string {
	r := make([]string, 0, len(t))
	for _, v := range t {
		r = append(r, v.Name)
	}
	return r
}

func (it *inflightTree) IsEmpty() bool {
	return (len(it.nonTrees) + len(it.trees)) == 0
}

func (it *inflightTree) UpdateFile(ctx context.Context, sourceStorer storer.Storer, targetStorer storer.Storer, hash plumbing.Hash, mode filemode.FileMode, filename string) error {
	it.nonTrees[filename] = object.TreeEntry{
		Mode: mode,
		Hash: hash,
		Name: filename,
	}

	logger.Debug("update file", "name", filename, "hash", hash.String())

	sourcefile, err := object.GetObject(sourceStorer, hash)
	if err != nil {
		return fmt.Errorf("failed to get non dir object at hash %s: %w", hash.String(), err)
	}

	targetfile := targetStorer.NewEncodedObject()
	if err := sourcefile.Encode(targetfile); err != nil {
		return fmt.Errorf("failed to encode file:%w ", err)
	}

	_, err = targetStorer.SetEncodedObject(targetfile)
	if err != nil {
		return fmt.Errorf("failed to save the object with hash %s: %w", hash.String(), err)
	}

	it.changed = true

	return nil
}

func (it *inflightTree) Update(ctx context.Context, sourceStorer storer.Storer, targetStorer storer.Storer, hash plumbing.Hash, mode filemode.FileMode, pathsegs []string) error {
	if len(pathsegs) == 0 {
		return fmt.Errorf("zero length path segment for file: %s", hash.String())
	}
	if len(pathsegs) == 1 {
		return it.UpdateFile(ctx, sourceStorer, targetStorer, hash, mode, pathsegs[0])
	}

	foldername := pathsegs[0]
	subtree, found := it.trees[foldername]
	var err error
	if !found {
		subtree, err = newInflightTree(&object.Tree{})
		if err != nil {
			return fmt.Errorf("failed to create an empty inflight tree: %w", err)
		}
		it.trees[foldername] = subtree
	}

	err = subtree.Update(ctx, sourceStorer, targetStorer, hash, mode, pathsegs[1:])
	if err != nil {
		return err
	}

	it.changed = true

	return nil
}

func (it *inflightTree) DeleteFile(ctx context.Context, hash plumbing.Hash, mode filemode.FileMode, filename string) error {
	filetodelete, found := it.nonTrees[filename]
	if !found {
		return fmt.Errorf("cannot find the file to delete: %s", filename)
	}

	if filetodelete.Hash != hash {
		return fmt.Errorf("hash of file in tree %s doesnt match the requested deletion %s", filetodelete.Hash.String(), hash.String())
	}
	if filetodelete.Mode != mode {
		return fmt.Errorf("mode of file %s doesnt match the requested deletion %s", filetodelete.Mode.String(), mode.String())
	}

	it.changed = true

	logger.Debug("delete file from tree", "file", filename)

	delete(it.nonTrees, filename)

	return nil
}

func (it *inflightTree) Delete(ctx context.Context, hash plumbing.Hash, mode filemode.FileMode, pathsegs []string) error {
	if len(pathsegs) == 0 {
		return fmt.Errorf("zero length path segment for file: %s", hash.String())
	}

	if len(pathsegs) == 1 {
		return it.DeleteFile(ctx, hash, mode, pathsegs[0])
	}

	foldername := pathsegs[0]
	subtree, found := it.trees[foldername]
	if !found {
		return fmt.Errorf("cannot find folder: %s", foldername)
	}

	err := subtree.Delete(ctx, hash, mode, pathsegs[1:])
	if err != nil {
		return errorf(err, "failed to delete %v: %w", strings.Join(pathsegs[1:], "/"), err)
	}
	if subtree.IsEmpty() {
		delete(it.trees, foldername)
	}

	it.changed = true

	return nil
}
