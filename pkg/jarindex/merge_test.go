package jarindex

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	jipb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/jarindex"
)

func TestMergeJarFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		predefined []string
		jars       []*jipb.JarFile
		wantWarn   string
		want       jipb.JarIndex
	}{
		"degenerate": {},
		"simple case": {
			jars: []*jipb.JarFile{
				{
					Label:    "//:foo",
					Filename: "foo.jar",
				},
				{
					Label:    "//:bar",
					Filename: "bar.jar",
				},
			},
			want: jipb.JarIndex{
				JarFile: []*jipb.JarFile{
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
			jars: []*jipb.JarFile{
				{Filename: "foo.jar"},
			},
			wantWarn: "missing jar label: foo.jar",
			want: jipb.JarIndex{
				JarFile: []*jipb.JarFile{
					{Filename: "foo.jar"},
				},
			},
		},
		"warns about missing filename": {
			jars: []*jipb.JarFile{
				{Label: "//:foo"},
			},
			wantWarn: "missing jar filename: //:foo",
			want: jipb.JarIndex{
				JarFile: []*jipb.JarFile{
					{Label: "//:foo"},
				},
			},
		},
		"warns about duplicate labels": {
			jars: []*jipb.JarFile{
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
			want: jipb.JarIndex{
				JarFile: []*jipb.JarFile{
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
