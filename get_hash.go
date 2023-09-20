package permgit

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetHash returns the hash of the
func GetHash(o object.Object) (*plumbing.Hash, error) {
	e := &plumbing.MemoryObject{}
	if err := o.Encode(e); err != nil {
		return nil, err
	}

	h := e.Hash()

	return &h, nil
}
