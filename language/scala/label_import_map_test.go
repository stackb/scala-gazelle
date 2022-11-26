package scala

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/index"
)

func TestLabelImportMapSet(t *testing.T) {
	io := NewDirectImportOrigin(&index.ScalaFileSpec{
		Filename: "Bar.scala",
	})
	io.Import = "com.foo.Bar"

	from := label.New("repo", "pkg", "name")

	lim := NewLabelImportMap()
	lim.Set(from, "com.foo.Bar", io)

	want := make(ImportOriginMap)
	want.Add("com.foo.Bar", io)

	got := lim[from]
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want +got):\n%s", diff)
	}
}
