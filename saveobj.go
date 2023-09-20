package permgit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// updateHashAndSave saves the [object.Object] without a hash value into the [storer.Storer], and decode it back to set the hash.
func updateHashAndSave(ctx context.Context, o object.Object, s storer.EncodedObjectStorer) error {
	ishashzero := o.ID().IsZero()
	if !ishashzero && s.HasEncodedObject(o.ID()) == nil {
		slog.Debug("object already in storage", "hash", o.ID().String())
		return nil
	}

	otype := o.Type().String()
	newstore := s.NewEncodedObject()
	if err := o.Encode(newstore); err != nil {
		return fmt.Errorf("failed to encode %s: %w", otype, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	hash, err := s.SetEncodedObject(newstore)
	if err != nil {
		return fmt.Errorf("failed to set %s: %w", otype, err)
	}

	if !ishashzero {
		return nil
	}
	saved, err := s.EncodedObject(o.Type(), hash)
	if err != nil {
		return fmt.Errorf("failed to read encoded %s back: %w", otype, err)
	}

	if err := o.Decode(saved); err != nil {
		return fmt.Errorf("failed to decode %s into proper format: %w", otype, err)
	}

	return nil
}
