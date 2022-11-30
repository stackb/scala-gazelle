package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

func TestLabelImportMapSet(t *testing.T) {
	io := NewDirectImportOrigin(&sppb.File{
		Filename: "Bar.scala",
	})
	io.Import = "com.foo.Bar"

	from := label.New("repo", "pkg", "name")

	lim := NewLabelImportMap()
	lim.Set(from, "com.foo.Bar", io)

	want := make(ImportOriginMap)
	want.Add("com.foo.Bar", io)

	got := lim[from]
	if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(sppb.File{})); diff != "" {
		t.Errorf("(-want +got):\n%s", diff)
	}
}
