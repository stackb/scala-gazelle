package scala

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestImportOriginString(t *testing.T) {
	for name, tc := range map[string]struct {
		origin  *ImportOrigin
		options []importOriginOption
		want    string
	}{
		"direct": {
			origin: NewDirectImportOrigin(&sppb.File{
				Filename: "Main.scala",
			}),
			want: "direct from Main.scala",
		},
		"direct with parent": {
			origin: NewDirectImportOrigin(&sppb.File{
				Filename: "Main.scala",
			}),
			options: []importOriginOption{
				withParent("com.foo.Bar"),
			},
			want: "direct from Main.scala (materialized from com.foo.Bar)",
		},
		"implicit": {
			origin: NewImplicitImportOrigin("com.foo.Bar"),
			want:   "implicit from com.foo.Bar",
		},
		"main_class": {
			origin: &ImportOrigin{Kind: ImportKindMainClass},
			want:   "main_class",
		},
		"comment": {
			origin: &ImportOrigin{Kind: ImportKindComment},
			want:   "comment",
		},
	} {
		t.Run(name, func(t *testing.T) {
			io := tc.origin
			for _, fn := range tc.options {
				io = fn(io)
			}
			got := io.String()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

type importOriginOption func(io *ImportOrigin) *ImportOrigin

func withParent(parent string) importOriginOption {
	return func(io *ImportOrigin) *ImportOrigin {
		io.Parent = parent
		return io
	}
}

func TestImportOriginMapAdd(t *testing.T) {
	m := make(ImportOriginMap)
	want := NewImplicitImportOrigin("com.foo.Bar")
	m.Add("com.foo.Foo", want)
	got := m["com.foo.Foo"]
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want +got):\n%s", diff)
	}
}

func TestImportOriginMapKeys(t *testing.T) {
	m := make(ImportOriginMap)
	m.Add("a", NewImplicitImportOrigin("b"))
	m.Add("b", NewImplicitImportOrigin("c"))
	want := []string{"a", "b"}
	got := m.Keys()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want +got):\n%s", diff)
	}
}
