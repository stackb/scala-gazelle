package crossresolve

import (
	"flag"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
)

func TestScalaSourceCrossResolverIsLabelOwner(t *testing.T) {
	for name, tc := range map[string]struct {
		lang      string
		imports   map[label.Label][]string
		from      label.Label
		indexFunc func(from label.Label) (*rule.Rule, bool)
		want      bool
	}{
		"degenerate case": {},
		"managed label": {
			lang: scalaName,
			indexFunc: func(from label.Label) (*rule.Rule, bool) {
				want := label.New("", "example", "scala_lib")
				if from != want {
					return nil, false
				}
				r := rule.NewRule("scala_library", "scala_lib")
				return r, true
			},
			from: label.New("", "example", "scala_lib"),
			want: true,
		},
		"unmanaged label": {
			lang: scalaName,
			from: label.New("", "example", "java_lib"),
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.indexFunc == nil {
				tc.indexFunc = func(from label.Label) (*rule.Rule, bool) {
					return nil, false
				}
			}
			args := []string{}

			cr := NewScalaSourceCrossResolver(tc.lang, func(src, dst, kind string) {})
			fs := flag.NewFlagSet(tc.lang, flag.ExitOnError)
			c := &config.Config{}
			cr.RegisterFlags(fs, cmdGenerate, c)
			if err := fs.Parse(args); err != nil {
				t.Fatal(err)
			}
			if err := cr.CheckFlags(fs, c); err != nil {
				t.Fatal(err)
			}

			cr.OnResolve()

			got := cr.IsLabelOwner(tc.from, tc.indexFunc)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".IsLabelOwner (-want +got):\n%s", diff)
			}
		})
	}
}
