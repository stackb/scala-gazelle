package crossresolve

import (
	"flag"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/google/go-cmp/cmp"
)

func TestProtoCrossResolverCrossResolve(t *testing.T) {
	for name, tc := range map[string]struct {
		lang    string
		imports map[label.Label][]string
		imp     resolve.ImportSpec
		want    []resolve.FindResult
	}{
		"hit": {
			lang: scalaName,
			imp:  resolve.ImportSpec{Lang: scalaName, Imp: "com.foo.proto.Customer"},
			imports: map[label.Label][]string{
				{Pkg: "com/foo/proto", Name: "customer_scala_proto"}: {"com.foo.proto.Customer"},
			},
			want: []resolve.FindResult{
				{Label: label.New("", "com/foo/proto", "customer_scala_proto")},
			},
		},
		"miss": {
			lang: scalaName,
			imp:  resolve.ImportSpec{Lang: scalaName, Imp: "com.foo.proto.Person"},
			imports: map[label.Label][]string{
				{Pkg: "com/foo/proto", Name: "customer_scala_proto"}: {"com.foo.proto.Customer"},
			},
			want: nil,
		},
		"name mismtch": {
			lang: "groovy",
			imp:  resolve.ImportSpec{Lang: scalaName, Imp: "com.foo.proto.Customer"},
			imports: map[label.Label][]string{
				{Pkg: "com/foo/proto", Name: "customer_scala_proto"}: {"com.foo.proto.Customer"},
			},
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			args := []string{}

			cr := NewProtoResolver(tc.lang, func(lang, impLang string) map[label.Label][]string {
				return tc.imports
			})
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

			mrslv := func(r *rule.Rule, pkgRel string) resolve.Resolver { return nil }
			ix := resolve.NewRuleIndex(mrslv)
			got := cr.CrossResolve(c, ix, tc.imp, scalaName)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf(".CrossResolve (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProtoCrossResolverIsLabelOwner(t *testing.T) {
	for name, tc := range map[string]struct {
		lang      string
		imports   map[label.Label][]string
		from      label.Label
		indexFunc func(from label.Label) (*rule.Rule, bool)
		want      bool
	}{
		"degenerate case": {},
		"managed proto label": {
			lang: scalaName,
			from: label.New("", "example", "foo_proto_scala_library"),
			want: true,
		},
		"managed grpc label": {
			lang: scalaName,
			from: label.New("", "example", "foo_grpc_scala_library"),
			want: true,
		},
		"unmanaged non-proto/non-grpc label": {
			lang: scalaName,
			from: label.New("", "example", "foo_scala_library"),
			want: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			args := []string{}

			cr := NewProtoResolver(tc.lang, func(lang, impLang string) map[label.Label][]string {
				return tc.imports
			})
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
