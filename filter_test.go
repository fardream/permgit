package permgit_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/fardream/permgit"
)

func TestPatternFilter_stringSplit(t *testing.T) {
	patterns := []struct {
		s    string
		want []string
	}{
		{"a/b/c", []string{"a", "b", "c"}},
		{"a/b/c/", []string{"a", "b", "c", ""}},
		{"/", []string{"", ""}},
	}

	for _, p := range patterns {
		x := strings.Split(p.s, "/")
		if !cmp.Equal(x, p.want) {
			t.Errorf("split %s, want: %v, got: %v", p.s, p.want, x)
		}
	}
}

func TestPatternFilter_Filter(t *testing.T) {
	patterns := []string{"aptos/**/*.js"}
	lines := []struct {
		name  string
		isdir bool
		want  []permgit.FilterResult
	}{
		{
			"aptos/test/test.js",
			false,
			[]permgit.FilterResult{permgit.FilterResult_In},
		},
	}

	filters := make([]*permgit.PatternFilter, 0, len(patterns))

	for _, p := range patterns {
		f, err := permgit.NewPatternFilter(p)
		if err != nil {
			t.Fatal(err)
		}
		filters = append(filters, f)
	}

	for _, l := range lines {
		segs := strings.Split(l.name, "/")
		for i, f := range filters {
			r := f.Filter(segs, l.isdir)
			if r != l.want[i] {
				t.Errorf("matching %s against %s, want %s, got %s", patterns[i], l.name, l.want[i].String(), r.String())
			}
		}
	}
}
