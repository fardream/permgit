package permgit_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/google/go-cmp/cmp"

	"github.com/fardream/permgit"
)

//go:embed testdata/testfilenames.txt
var testfilenames string

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
	patterns := []string{"aptos/**/*.js", "aptos/**/src"}
	lines := []struct {
		name  string
		isdir bool
		want  []permgit.FilterResult
	}{
		{
			"aptos/test/test.js",
			false,
			[]permgit.FilterResult{permgit.FilterResult_In, permgit.FilterResult_Out},
		},
		{
			"aptos/test/src/abc.rs",
			false,
			[]permgit.FilterResult{permgit.FilterResult_Out, permgit.FilterResult_In},
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

func TestPatternFilter_Filter_many(t *testing.T) {
	filelines := strings.Split(testfilenames, "\n")
	patterns := []string{
		"/aptos/**/*.js",
		"/aptos/**/src/",
		"/aptos/**/lib.rs",
		"/LICENSE",
		"/LICENSE_*",
	}
	for _, ap := range patterns {
		f, err := permgit.NewPatternFilter(ap)
		if err != nil {
			t.Error(err)
		}
		gp := gitignore.ParsePattern(ap, nil)

		for _, line := range filelines {
			paths := strings.Split(line, "/")
			r := gp.Match(paths, false)
			fr := f.Filter(paths, false)
			if r == gitignore.Exclude && fr != permgit.FilterResult_In {
				t.Errorf("gitignore says %s but we says %s for pattern %s and line %s", "in", fr.String(), ap, line)
			} else if r == gitignore.NoMatch && fr != permgit.FilterResult_Out {
				t.Errorf("gitignore says %s but we says %s for pattern %s and line %s", "out", fr.String(), ap, line)
			}
		}
	}
}
