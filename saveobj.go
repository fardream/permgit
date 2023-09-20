package permgit

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// updateHashAndSave saves the [object.Object] without a hash value into the [storer.Storer], and decode it back to set the hash.
func updateHashAndSave(o object.Object, s storer.EncodedObjectStorer) error {
	otype := o.Type().String()
	newstore := s.NewEncodedObject()
	if err := o.Encode(newstore); err != nil {
		return fmt.Errorf("failed to encode %s: %w", otype, err)
	}

	h, err := s.SetEncodedObject(newstore)
	if err != nil {
		return fmt.Errorf("failed to set %s: %w", otype, err)
	}

	saved, err := s.EncodedObject(o.Type(), h)
	if err != nil {
		return fmt.Errorf("failed to read encoded %s back: %w", otype, err)
	}

	if err := o.Decode(saved); err != nil {
		return fmt.Errorf("failed to decode %s into proper format: %w", otype, err)
	}

	return nil
}
