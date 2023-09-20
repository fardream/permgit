package permgit

import "strings"

// TreeEntryFilter is the interface used to filter the path of the tree.
type TreeEntryFilter interface {
	IsIn(p string) bool
}

// PrefixFilter filters the entry of the tree by compare it with prefix. If the full path of the entry starts with the prefix, it will be considered in.
type PrefixFilter struct {
	prefix string
}

var _ TreeEntryFilter = (*PrefixFilter)(nil)

func (f *PrefixFilter) IsIn(p string) bool {
	return strings.HasPrefix(p, f.prefix)
}

func NewPrefixFilter(prefix string) *PrefixFilter {
	return &PrefixFilter{prefix: prefix}
}

var _ TreeEntryFilter = (*AndFilter)(nil)

// AndFilter combines multiple [TreeEntryFilter] into one [TreeEntryFilter] with an "and" operation, the path will only be included when all the filters include it.
type AndFilter struct {
	filters []TreeEntryFilter
}

func (f *AndFilter) IsIn(p string) bool {
	if len(f.filters) == 0 {
		return false
	}

	n := len(f.filters)
	in := f.filters[0].IsIn(p)
	for i := 1; i < n; i++ {
		if !in {
			break
		}
		in = in && f.filters[n].IsIn(p)
	}

	return in
}

func (f *AndFilter) Add(filters ...TreeEntryFilter) {
	f.filters = append(f.filters, filters...)
}

// NewAndFilter creates a new filter with and operations.
func NewAndFilter(filters ...TreeEntryFilter) *AndFilter {
	f := &AndFilter{}
	f.Add(filters...)

	return f
}

// OrFilter combines multiple [TreeEntryFilter] into one [TreeEntryFilter] with an "or" operation, the path will be inclueded if any one of the filters includes it.
type OrFilter struct {
	filters []TreeEntryFilter
}

var _ TreeEntryFilter = (*OrFilter)(nil)

func (f *OrFilter) IsIn(p string) bool {
	for _, sf := range f.filters {
		if sf.IsIn(p) {
			return true
		}
	}

	return false
}

func (f *OrFilter) Add(filters ...TreeEntryFilter) {
	f.filters = append(f.filters, filters...)
}

func NewOrFilter(filters ...TreeEntryFilter) *OrFilter {
	f := &OrFilter{}

	f.Add(filters...)

	return f
}
