package permgit

import "log/slog"

var logger = slog.Default()

func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}
