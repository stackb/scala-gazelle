package mergeindex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/golang/protobuf/proto"
	"github.com/stackb/scala-gazelle/api/jarindex"
)

func TestReadWriteJarIndexProtoFile(t *testing.T) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.proto.jarindex")
	want := &jarindex.JarIndex{
		JarFile: []*jarindex.JarFile{
			{Filename: "foo.jar"},
			{Filename: "bar.jar"},
		},
	}
	if err := WriteJarIndexProtoFile(filename, want); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadJarIndexProtoFile(filename); err != nil {
		t.Fatal(err)
	} else {
		if !proto.Equal(want, got) {
			t.Fatalf("jarindex read/write symmetry error (want=%+v, got=%+v)", want, got)
		}
	}
}
