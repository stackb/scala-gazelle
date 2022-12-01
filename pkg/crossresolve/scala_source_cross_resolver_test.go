package crossresolve

import (
	"flag"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func TestScalaSourceCrossResolverConfigureCacheFile(t *testing.T) {
	for name, tc := range map[string]struct {
		args          []string
		wantCacheFile string
		wantErr       error
	}{
		"degenerate case": {},
	} {
		t.Run(name, func(t *testing.T) {
			cr := NewScalaSourceCrossResolver(scalaName)
			fs := flag.NewFlagSet(scalaName, flag.ExitOnError)
			c := &config.Config{}
			cr.RegisterFlags(fs, cmdGenerate, c)

			if testutil.ExpectError(t, tc.wantErr, fs.Parse(tc.args)) {
				return
			}

			//
			// TODO: why does this just hang the test?
			//

			// if testutil.ExpectError(t, tc.wantErr, cr.CheckFlags(fs, c)) {
			// 	return
			// }

			// if diff := cmp.Diff(tc.wantCacheFile, cr.cacheFile); diff != "" {
			// 	t.Errorf(".cacheFile (-want +got):\n%s", diff)
			// }
		})
	}
}

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

			cr := NewScalaSourceCrossResolver(tc.lang)
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
