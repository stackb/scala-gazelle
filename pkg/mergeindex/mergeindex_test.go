package mergeindex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/api/jarindex"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
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

func TestReadWriteJarFileProtoFile(t *testing.T) {
	tmpDir, err := bazel.NewTmpDir("")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filename := filepath.Join(tmpDir, "test.proto.jarfile")
	want := &jarindex.JarFile{
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

func TestMergeJarFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		predefined []string
		jars       []*jarindex.JarFile
		wantWarn   string
		want       jarindex.JarIndex
	}{
		"degenerate": {},
		"simple case": {
			jars: []*jarindex.JarFile{
				{
					Label:    "//:foo",
					Filename: "foo.jar",
				},
				{
					Label:    "//:bar",
					Filename: "bar.jar",
				},
			},
			want: jarindex.JarIndex{
				JarFile: []*jarindex.JarFile{
					{
						Label:    "//:foo",
						Filename: "foo.jar",
					},
					{
						Label:    "//:bar",
						Filename: "bar.jar",
					},
				},
			},
		},
		"warns about missing label": {
			jars: []*jarindex.JarFile{
				{
					Filename: "foo.jar",
				},
			},
			wantWarn: "missing jar label: foo.jar",
			want: jarindex.JarIndex{
				JarFile: []*jarindex.JarFile{},
			},
		},
		"warns about missing filename": {
			jars: []*jarindex.JarFile{
				{
					Label: "//:foo",
				},
			},
			wantWarn: "missing jar filename: //:foo",
			want: jarindex.JarIndex{
				JarFile: []*jarindex.JarFile{},
			},
		},
		"warns about duplicate labels": {
			jars: []*jarindex.JarFile{
				{
					Label:    "//:foo",
					Filename: "foo.jar",
				},
				{
					Label:    "//:foo",
					Filename: "foo.jar",
				},
			},
			wantWarn: "duplicate jar label: //:foo",
			want: jarindex.JarIndex{
				JarFile: []*jarindex.JarFile{
					{
						Label:    "//:foo",
						Filename: "foo.jar",
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var gotWarn strings.Builder
			got, err := MergeJarFiles(func(format string, args ...interface{}) {
				gotWarn.WriteString(fmt.Sprintf(format, args...))
			}, tc.predefined, tc.jars)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.wantWarn, gotWarn.String()); diff != "" {
				t.Errorf("merge warning (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(&tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("index diff (-want +got):\n%s", diff)
			}
		})
	}
}
