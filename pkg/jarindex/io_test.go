package jarindex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"google.golang.org/protobuf/proto"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
)

func TestReadWriteJarIndexProtoFile(t *testing.T) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.proto.jarindex")
	want := &jipb.JarIndex{
		JarFile: []*jipb.JarFile{
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

func TestReadWriteJarFileProtoFile(t *testing.T) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.proto.jarfile")
	want := &jipb.JarFile{
		Filename: "foo.jar",
	}
	if err := WriteJarFileProtoFile(filename, want); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadJarFileProtoFile(filename); err != nil {
		t.Fatal(err)
	} else {
		if !proto.Equal(want, got) {
			t.Fatalf("jarfile read/write symmetry error (want=%+v, got=%+v)", want, got)
		}
	}
}
