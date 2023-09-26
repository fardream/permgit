package permgit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

func getFileOperation(fromfilename string, tofilename string) string {
	switch {
	case fromfilename == "":
		return "add"
	case tofilename == "":
		return "delete"
	case fromfilename != tofilename:
		return "rename"
	default:
		return "modify"
	}
}

// FilePatchError is an error containing the information about the invalid file patch.
type FilePatchError struct {
	FromFile string
	ToError  string
}

func (e *FilePatchError) Error() string {
	errfs := make([]string, 0, 2)
	if e.FromFile != "" {
		errfs = append(errfs, fmt.Sprintf("invalid from path: %s", e.FromFile))
	}
	if e.ToError != "" {
		errfs = append(errfs, fmt.Sprintf("invalid to path: %s", e.ToError))
	}

	return strings.Join(errfs, "|")
}

// ExpandTree apply the changes made in the filteredNew tree to filteredOrig tree and apply them to target tree, it returns a new tree.
func ExpandTree(
	ctx context.Context,
	sourceStorer storer.Storer,
	filteredOrig *object.Tree,
	filteredNew *object.Tree,
	target *object.Tree,
	targetStorer storer.Storer,
	filter Filter,
) (*object.Tree, error) {
	filteredPath, err := filteredOrig.Patch(filteredNew)
	if err != nil {
		return nil, fmt.Errorf("failed to generate path for the two filtered trees: %w", err)
	}

	filepatches := filteredPath.FilePatches()

	// collect all invalid file paths into the errors
	var errs []error

	// first pass, check if the patches are valid.
	for i, afile := range filepatches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		fromfile, tofile := afile.Files()

		fromfilename := ""
		if fromfile != nil {
			fromfilename = fromfile.Path()
		}
		tofilename := ""
		if tofile != nil {
			tofilename = tofile.Path()
		}

		logger.Debug("patch", "idx", i, "operation", getFileOperation(fromfilename, tofilename), "from", fromfilename, "to", tofilename)

		var thiserr *FilePatchError
		if fromfile != nil && !filter.Filter(strings.Split(fromfilename, "/"), false).IsIn() {
			if thiserr == nil {
				thiserr = new(FilePatchError)
			}
			thiserr.FromFile = fromfilename
		}
		if tofile != nil && !filter.Filter(strings.Split(tofilename, "/"), false).IsIn() {
			if thiserr == nil {
				thiserr = new(FilePatchError)
			}
			thiserr.ToError = tofilename
		}
		if thiserr != nil {
			errs = append(errs, thiserr)
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	editTree, err := newInflightTree(target)
	if err != nil {
		return nil, err
	}

	// second pass, delete files that are deleted or renamed
	for _, afile := range filepatches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		fromfile, tofile := afile.Files()
		if !(fromfile != nil && (tofile == nil || tofile.Path() != fromfile.Path())) {
			continue
		}
		if fromfile.Mode() == filemode.Submodule {
			logger.Warn("silently ignore submodule in from-file", "path", fromfile.Path())
			continue
		}

		err := editTree.Delete(ctx, fromfile.Hash(), fromfile.Mode(), strings.Split(fromfile.Path(), "/"))
		if err != nil {
			return nil, errorf(err, "failed to delete file %s: %w", fromfile.Path(), err)
		}
	}

	// third pass, update files (new, renamed, or modified)
	for _, afile := range filepatches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, tofile := afile.Files()
		if tofile == nil {
			continue
		}
		if tofile.Mode() == filemode.Submodule {
			logger.Warn("silently ignore submodule in to-file", "path", tofile.Path())
			continue
		}

		if err := editTree.Update(ctx, sourceStorer, targetStorer, tofile.Hash(), tofile.Mode(), strings.Split(tofile.Path(), "/")); err != nil {
			return nil, errorf(err, "failed to update file %s %s: %w", tofile.Path(), tofile.Hash(), err)
		}

	}

	newtree, err := editTree.BuildTree(ctx, targetStorer)
	if err != nil {
		return nil, err
	}

	return newtree, nil
}
