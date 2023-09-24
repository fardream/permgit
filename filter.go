package permgit

import (
	"slices"
	"strings"
)

// FilterResult indicates the result of a filter, it can be
//   - the input is out
//   - the input is directory, and its entries should be filtered
//   - the input is in
//
// the logic or operation for filter result:
//   - If out and in, in
//   - If out and dir_dive, dir_dive
//   - If dir_dive and in, in
//
// the logic and operation for filter result:
//   - if out and int, out
//   - if out and dir_dive, out
//   - if dir_dive and in, dir_dive
//
// Notice that the enum values has [FilterResult_In] at 2, [FilterResult_DirDive] at 1, and [FilterResult_Out] at 0,
// therefore the or operation is finding the max, and and operation is finding the min.
type FilterResult uint8

const (
	FilterResult_Out     FilterResult = iota // Out
	FilterResult_DirDive                     // DirDive
	FilterResult_In                          // In
)

// If the filter result is in
func (r FilterResult) IsIn() bool {
	return r == FilterResult_In
}

// FilterResultsOr perform or operation on filter results:
//   - If out and in, in
//   - If out and dir_dive, dir_dive
//   - If dir_dive and in, in
//
// This is equivalent to take the max value of the input.
func FilterResultsOr(r ...FilterResult) FilterResult {
	return slices.Max(r)
}

// FilterResultsAnd perform and operation on the filter results:
//   - if out and int, out
//   - if out and dir_dive, out
//   - if dir_dive and in, dir_dive
func FilterResultsAnd(r ...FilterResult) FilterResult {
	return slices.Min(r)
}

// Filter is the interface used to filter the path of the tree.
type Filter interface {
	Filter(paths []string, isdir bool) FilterResult
}

// FilterPath calls [Filter] f on fullpath string.
func FilterPath(f Filter, fullpath string, isdir bool) FilterResult {
	return f.Filter(strings.Split(fullpath, "/"), isdir)
}
