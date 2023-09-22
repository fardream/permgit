package permgit

import (
	"context"
	"errors"
	"fmt"
)

// errorf checks if the given error is [context.Canceled] or [context.DeadelineExceeded], and
// returns the error unmodified if it is so, otherwise call [fmt.errorf] with the provided parameters.
func errorf(err error, format string, a ...any) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return fmt.Errorf(format, a...)
}
