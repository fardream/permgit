package permgit

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func DumpTree(ctx context.Context, prepath []string, tree *object.Tree, filter Filter, output io.Writer) error {
	for _, v := range tree.Entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		fullpath := addpath(prepath, v.Name)
		switch v.Mode {
		case filemode.Dir:
			if filter.Filter(fullpath, true) == FilterResult_Out {
				continue
			}
			subtree, err := tree.Tree(v.Name)
			if err != nil {
				return fmt.Errorf("failed to obtain tree %s: %w", fullpath, err)
			}

			err = DumpTree(ctx, fullpath, subtree, filter, output)
			if err != nil {
				return errorf(err, "failed to dump tree %s: %w", fullpath, err)
			}

		default:
			if filter.Filter(fullpath, false) == FilterResult_Out {
				continue
			}
			fmt.Fprintln(output, strings.Join(fullpath, "/"))
		}
	}

	return nil
}
