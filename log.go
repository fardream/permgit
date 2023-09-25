package permgit

import "log/slog"

var logger = slog.Default()

// SetLogger sets the logger used by permgit.
// The default one comes from [slog.Default].
func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}
