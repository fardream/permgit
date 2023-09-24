package permgit

// TrueFilter always return [FilterResult_In] for any input.
type TrueFilter struct{}

var _ Filter = (*TrueFilter)(nil)

func (TrueFilter) Filter(path []string, isdir bool) FilterResult {
	return FilterResult_In
}
