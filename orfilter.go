package permgit

// OrFilter combines multiple [Filter] into one [Filter] with an "or" operation, the path will be inclueded if any one of the filters includes it.
type OrFilter struct {
	filters []Filter
}

var _ Filter = (*OrFilter)(nil)

func (f *OrFilter) Filter(paths []string, isdir bool) FilterResult {
	if len(f.filters) == 0 {
		return FilterResult_Out
	}

	n := len(f.filters)
	in := f.filters[0].Filter(paths, isdir)
	for i := 1; i < n; i++ {
		if in == FilterResult_In {
			break
		}
		in = max(in, f.filters[i].Filter(paths, isdir))
	}

	return in
}

func (f *OrFilter) Add(filters ...Filter) {
	f.filters = append(f.filters, filters...)
}

func NewOrFilter(filters ...Filter) *OrFilter {
	f := &OrFilter{}

	f.Add(filters...)

	return f
}

// NewOrFilterForPatterns creates a new Or filter for all the patterns
func NewOrFilterForPatterns(patterns ...string) (Filter, error) {
	r := &OrFilter{
		filters: make([]Filter, 0, len(patterns)),
	}

	for _, v := range patterns {
		p, err := NewPatternFilter(v)
		if err != nil {
			return nil, err
		}
		r.filters = append(r.filters, p)
	}

	return NewCachedFilter(r), nil
}
