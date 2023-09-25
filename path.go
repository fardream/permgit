package permgit

import "strings"

func addpath(prefix []string, name string) []string {
	return append(prefix[:], name)
}

func pathsToFullPath(paths []string) string {
	return strings.Join(paths, "/")
}
