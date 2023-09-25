package permgit

import (
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// DumpTree writes the file entries in this tree and its sub trees to an [io.Writer].
func DumpTree(ctx context.Context, prepath []string, tree *object.Tree, filter Filter, output io.Writer) error {
	for _, v := range tree.Entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		fullpath := addpath(prepath, v.Name)
		fullpathstring := pathsToFullPath(fullpath)
		switch v.Mode {
		case filemode.Dir:
			if filter.Filter(fullpath, true) == FilterResult_Out {
				continue
			}
			subtree, err := tree.Tree(v.Name)
			if err != nil {
				return fmt.Errorf("failed to obtain tree %s: %w", fullpathstring, err)
			}

			err = DumpTree(ctx, fullpath, subtree, filter, output)
			if err != nil {
				return errorf(err, "failed to dump tree %s: %w", fullpathstring, err)
			}

		default:
			if filter.Filter(fullpath, false) == FilterResult_Out {
				continue
			}
			fmt.Fprintln(output, fullpathstring)
		}
	}

	return nil
}
