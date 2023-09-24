package permgit

// AndFilter combines multiple [TreeEntryFilter] into one [TreeEntryFilter] with an "and" operation, the path will only be included when all the filters include it.
type AndFilter struct {
	filters []Filter
}

var _ Filter = (*AndFilter)(nil)

func (f *AndFilter) Filter(paths []string, isdir bool) FilterResult {
	if len(f.filters) == 0 {
		return FilterResult_Out
	}

	n := len(f.filters)
	in := f.filters[0].Filter(paths, isdir)
	for i := 1; i < n; i++ {
		if in == FilterResult_Out {
			break
		}
		in = min(in, f.filters[i].Filter(paths, isdir))
	}

	return in
}

func (f *AndFilter) Add(filters ...Filter) {
	f.filters = append(f.filters, filters...)
}

// NewAndFilter creates a new filter with and operations.
func NewAndFilter(filters ...Filter) *AndFilter {
	f := &AndFilter{}
	f.Add(filters...)

	return f
}
