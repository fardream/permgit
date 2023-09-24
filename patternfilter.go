package permgit

import (
	"fmt"
	"path"
	"strings"
)

// PatternFilter filters the entries according to a restricted pattern of .gitignore
//
//   - '**' is for multi level directories, and it can only appear once in the match.
//   - '*' is for match one level of names.
//   - '!' and escapes are unsupported.
type PatternFilter struct {
	inputPattern    string
	filterSegments  []string
	multiLevelIndex int
	// isDirOnly indicates if the filter is for directories only.
	// this is false indicating this matches files and directories.
	isDirOnly bool

	beforefilters []string
	afterfilters  []string
}

var _ Filter = (*PatternFilter)(nil)

func NewPatternFilter(pattern string) (*PatternFilter, error) {
	trimmedpattern := strings.TrimSpace(pattern)
	p := &PatternFilter{
		inputPattern:    trimmedpattern,
		multiLevelIndex: -1,
	}

	// remove trailing **/ or **
	if strings.HasSuffix(trimmedpattern, "**/") {
		trimmedpattern = strings.TrimSuffix(trimmedpattern, "**/")
	} else {
		// if strings.HasSuffix(trimmedpattern, "**") is unnecessary
		trimmedpattern = strings.TrimSuffix(trimmedpattern, "**")
	}

	logger.Debug("pattern", "input", pattern, "trimmed", trimmedpattern)

	if trimmedpattern == "/" || trimmedpattern == "" {
		return nil, fmt.Errorf("'%s' is invalid pattern", trimmedpattern)
	}

	p.isDirOnly = strings.HasSuffix(p.inputPattern, "/")
	p.filterSegments = strings.Split(p.inputPattern, "/")
	if len(p.filterSegments) == 0 {
		return nil, fmt.Errorf("input pattern %s has zero path segments", pattern)
	}

	// if the pattern ends with /, the last segment is empty
	if p.isDirOnly {
		// even after this, the segs should still has at least 1 element
		p.filterSegments = p.filterSegments[:len(p.filterSegments)-1]
	}

	// if the first element is empty, there is a root / at the start of the pattern.
	if p.filterSegments[0] == "" {
		p.filterSegments = p.filterSegments[1:]
	}

	if len(p.filterSegments) == 0 {
		return nil, fmt.Errorf("zero path segment left after removing leading/trailing white spaces: '%s'", trimmedpattern)
	}

	for idx, seg := range p.filterSegments {
		// check on the segment
		if seg == "**" {
			if p.multiLevelIndex >= 0 {
				return nil, fmt.Errorf("at most 1 ** pattern can appear in pattern, but %s has more than 1", trimmedpattern)
			}
			if idx == len(p.filterSegments)-1 {
				return nil, fmt.Errorf("trailing ** or **/ hasn't been removed")
			}
			p.multiLevelIndex = idx
		} else if strings.Contains(seg, "**") {
			return nil, fmt.Errorf("segment: %s contains **, which is invalid", seg)
		} else {
			_, err := path.Match(seg, "abc")
			if err != nil {
				return nil, fmt.Errorf("pattern segment %s is not valid: %w", seg, err)
			}
		}
	}

	if p.multiLevelIndex >= 0 {
		p.beforefilters = p.filterSegments[:p.multiLevelIndex]
		p.afterfilters = []string{}
		if len(p.filterSegments) > p.multiLevelIndex+1 {
			p.afterfilters = p.filterSegments[p.multiLevelIndex+1:]
		}
		logger.Debug("multi-level-filter", "before", p.beforefilters, "after", p.afterfilters)
	}

	return p, nil
}

func (f *PatternFilter) Filter(paths []string, isdir bool) FilterResult {
	if f.multiLevelIndex < 0 {
		// not multiLevelIndex, use simple filter
		return nonMultiLevelFilter(isdir, paths, f.filterSegments, f.isDirOnly)
	}

	beforefilters := f.beforefilters
	afterfilters := f.afterfilters

	predirpaths := paths[:]
	if !isdir {
		predirpaths = predirpaths[:len(predirpaths)-1]
	}

	if len(predirpaths) >= f.multiLevelIndex {
		predirpaths = predirpaths[:f.multiLevelIndex]
	}

	remainingpaths := paths[len(predirpaths):]

	beforesult := DirFilter(predirpaths, beforefilters)
	switch beforesult {
	case FilterResult_In:
		if len(afterfilters) == 0 {
			return FilterResult_In
		}
		if len(remainingpaths) == 0 {
			return FilterResult_DirDive
		}
		r := FilterResult_Out
		for i := 0; i < len(remainingpaths); i++ {
			afterpaths := remainingpaths[i:]
			tr := nonMultiLevelFilter(isdir, afterpaths, afterfilters, f.isDirOnly)
			if tr > r {
				r = tr
			}
			if r == FilterResult_In {
				return r
			}
		}

		if r == FilterResult_Out && isdir {
			return FilterResult_DirDive
		}

		return r
	case FilterResult_DirDive:
		if !isdir || len(remainingpaths) > 0 {
			return FilterResult_Out
		}
		return FilterResult_DirDive
	case FilterResult_Out:
		fallthrough
	default:
		return FilterResult_Out
	}
}

func nonMultiLevelFilter(isdir bool, paths []string, filters []string, fisdir bool) FilterResult {
	switch {
	case isdir:
		// input is dir, do DirFilter
		return DirFilter(paths, filters)
	case !isdir && fisdir:
		// input is a file, so it will only be in if its dir is in
		if DirFilter(paths[:len(paths)-1], filters) != FilterResult_In {
			return FilterResult_Out
		} else {
			return FilterResult_In
		}
	case !isdir && !fisdir:
		if len(paths) < len(filters) {
			return FilterResult_Out
		}

		return DirFilter(paths, filters)
	default:
		return FilterResult_Out
	}
}

// DirFilter filters the directory according to a directory filter.
//   - In, if filters match all the leading path segments, and there are zero or more path trailing.
//     | p | p | p | p
//     | f | f | f
//     Or
//     | p | p | p
//     | f | f | f
//   - DirDive, if size of path segments is smaller than filters, and those path segments match the corresponding filters
//     | p | p | p
//     | f | f | f | f
//   - otherwise it's out.
func DirFilter(paths []string, filtersegs []string) FilterResult {
	if len(paths) == 0 || len(filtersegs) == 0 {
		return FilterResult_Out
	}

	if len(paths) >= len(filtersegs) {
		for i, fseg := range filtersegs {
			matched, err := path.Match(fseg, paths[i])
			if err != nil {
				logger.Warn("failed match", "pattern", fseg, "name", paths[i], "error", err.Error())
				return FilterResult_Out
			}
			if !matched {
				return FilterResult_Out
			}
		}

		return FilterResult_In
	} else {
		for i, p := range paths {
			matched, err := path.Match(filtersegs[i], p)
			if err != nil {
				logger.Warn("failed match", "pattern", filtersegs[i], "name", p, "error", err.Error())
				return FilterResult_Out
			}
			if !matched {
				return FilterResult_Out
			}
		}

		return FilterResult_DirDive
	}
}
