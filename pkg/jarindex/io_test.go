package jarindex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"google.golang.org/protobuf/proto"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
)

func TestReadWriteJarIndexFile(t *testing.T) {
	for name, tc := range map[string]struct {
		filename string
	}{
		"proto": {
			filename: "test.javaindex.pb",
		},
		"json": {
			filename: "test.javaindex.json",
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmpDir, err := bazel.NewTmpDir("")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			filename := filepath.Join(tmpDir, tc.filename)
			want := &jipb.JarIndex{
				JarFile: []*jipb.JarFile{
					{Filename: "foo.jar"},
					{Filename: "bar.jar"},
				},
			}
			if err := WriteJarIndexFile(filename, want); err != nil {
				t.Fatal(err)
			}
			if got, err := ReadJarIndexFile(filename); err != nil {
				t.Fatal(err)
			} else {
				if !proto.Equal(want, got) {
					t.Fatalf("jarindex read/write symmetry error (want=%+v, got=%+v)", want, got)
				}
			}
		})
	}
}
