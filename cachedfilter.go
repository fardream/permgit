package permgit

import "strings"

// CachedFilter records the paths it sees - the cache is no concurrent safe.
type CachedFilter struct {
	filter Filter

	dircache    map[string]FilterResult
	nondircache map[string]FilterResult
}

var _ Filter = (*CachedFilter)(nil)

func (f *CachedFilter) Filter(paths []string, isdir bool) FilterResult {
	name := strings.Join(paths, "/")
	if isdir {
		r, in := f.dircache[name]
		if in {
			return r
		}

		r = f.filter.Filter(paths, isdir)
		f.dircache[name] = r
		return r
	} else {
		r, in := f.nondircache[name]
		if in {
			return r
		}

		r = f.filter.Filter(paths, isdir)
		f.nondircache[name] = r
		return r
	}
}

func NewCachedFilter(underlying Filter) *CachedFilter {
	return &CachedFilter{
		filter:      underlying,
		dircache:    make(map[string]FilterResult),
		nondircache: make(map[string]FilterResult),
	}
}

// Reset clears up the cache
func (f *CachedFilter) Reset() {
	clear(f.dircache)
	clear(f.nondircache)
}
