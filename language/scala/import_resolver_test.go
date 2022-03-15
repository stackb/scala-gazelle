package scala

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/index"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/resolve"
)

// TestImportRegistryCompletions tests the parsing of a starlark glob.
func TestImportRegistryCompletions(t *testing.T) {
	for name, tc := range map[string]struct {
		registry *importRegistry
		imp      string
		want     map[string]label.Label
	}{
		"empty": {
			want:     make(map[string]label.Label),
			registry: newImportRegistryBuilder(t).build(),
		},
		"collect": {
			imp: "com.lib._",
			registry: newImportRegistryBuilder(t).
				provides("//com/lib:bar", "com.lib.A").
				provides("//com/lib:baz", "com.lib.C", "com.lib.D").
				provides("//com/lib:xxx", "com.xxx.E", "com.xxx.F").
				build(),
			want: map[string]label.Label{
				"A": mustParseLabel(t, "//com/lib:bar"),
				"C": mustParseLabel(t, "//com/lib:baz"),
				"D": mustParseLabel(t, "//com/lib:baz"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := tc.registry.Completions(tc.imp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("importRegistry.completions (-want +got):\n%s", diff)
			}
		})
	}
}

// TestImportRegistryDisambiguate tests the parsing of a starlark glob.
func SkipTestImportRegistryDisambiguateErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		registry *importRegistry
		imp      string
		labels   []label.Label
		from     label.Label
		want     string
	}{
		"empty": {
			registry: newImportRegistryBuilder(t).build(),
			want:     `no completions known for //: (aborting disambiguation attempt of "")`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := &config.Config{}
			_, got := tc.registry.Disambiguate(c, nil, resolve.ImportSpec{Imp: tc.imp, Lang: ScalaLangName}, ScalaLangName, tc.from, tc.labels)
			if got == nil {
				t.Fatal("expected err, got none")
			}
			if diff := cmp.Diff(tc.want, got.Error()); diff != "" {
				t.Errorf("importRegistry.DisambiguateErrors (-want +got):\n%s", diff)
			}
		})
	}
}

// TestImportRegistryDisambiguate tests the parsing of a starlark glob.
func TestImportRegistryDisambiguate(t *testing.T) {
	for name, tc := range map[string]struct {
		registry *importRegistry
		imp      string
		labels   []label.Label
		from     label.Label
		want     []label.Label
	}{
		"success": {
			registry: newImportRegistryBuilder(t).
				provides("//com/lib:bar", "com.lib.A").
				provides("//com/lib:baz", "com.lib.C", "com.lib.D").
				sourceImports("//com/app:app", "com/app/App.scala", "com.lib._").
				notFoundTypes("com/app/App.scala", "D").
				build(),
			imp:    "com.lib._",
			labels: mustParseLabels(t, "//com/lib:bar", "//com/lib:baz"),
			from:   mustParseLabel(t, "//com/app:app"),
			want:   mustParseLabels(t, "//com/lib:baz"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := &config.Config{}
			got, err := tc.registry.Disambiguate(c, nil, resolve.ImportSpec{Imp: tc.imp, Lang: ScalaLangName}, ScalaLangName, tc.from, tc.labels)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("importRegistry.completions (-want +got):\n%s", diff)
			}
		})
	}
}

// TestImportRegistryTransitiveImports transitive imports calculation.
func TestImportRegistryTransitiveImports(t *testing.T) {
	for name, tc := range map[string]struct {
		registry *importRegistry
		imps     []string
		want     []string
	}{
		"degenerate": {
			registry: newImportRegistryBuilder(t).
				depends(map[string]string{}).
				build(),
		},
		"first-order deps resolve": {
			registry: newImportRegistryBuilder(t).
				depends(map[string]string{
					"a": "b",
					"b": "c",
					"c": "d",
				}).
				build(),
			imps: []string{"a"},
			want: []string{"b"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := tc.registry.TransitiveImports(tc.imps)
			if err != nil {
				t.Fatal("unexpected error:", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("importRegistry.TransitiveImports (-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseLabels(t *testing.T, lbls ...string) []label.Label {
	ll := make([]label.Label, len(lbls))
	for i, l := range lbls {
		ll[i] = mustParseLabel(t, l)
	}
	return ll
}

func mustParseLabel(t *testing.T, lbl string) label.Label {
	l, err := label.Parse(lbl)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

type importRegistryBuilder struct {
	t            *testing.T
	ruleRegistry *fakeRuleRegistry
	compiler     *fakeCompiler
	registry     *importRegistry
}

func newImportRegistryBuilder(t *testing.T) *importRegistryBuilder {
	rr := &fakeRuleRegistry{
		t:     t,
		rules: make(map[label.Label]*index.ScalaRuleSpec),
	}
	cr := &fakeClassRegistry{
		t: t,
	}

	c := &fakeCompiler{
		t:       t,
		results: make(map[string]*index.ScalaCompileSpec),
	}

	return &importRegistryBuilder{
		t:            t,
		ruleRegistry: rr,
		compiler:     c,
		registry:     newImportRegistry(rr, cr, c),
	}
}

func (b *importRegistryBuilder) sourceImports(from, filename string, imports ...string) *importRegistryBuilder {
	fromLabel, err := label.Parse(from)
	if err != nil {
		b.t.Fatal(err)
	}
	rule := b.ruleRegistry.rules[fromLabel]
	if rule == nil {
		rule = &index.ScalaRuleSpec{
			Srcs: make([]*index.ScalaFileSpec, 0),
		}
		b.ruleRegistry.rules[fromLabel] = rule
	}
	rule.Srcs = append(rule.Srcs, &index.ScalaFileSpec{
		Filename: filename,
		Imports:  imports,
	})
	return b
}

func (b *importRegistryBuilder) notFoundTypes(filename string, notFoundTypes ...string) *importRegistryBuilder {
	nf := make([]*index.NotFoundSymbol, len(notFoundTypes))
	for i, t := range notFoundTypes {
		nf[i] = &index.NotFoundSymbol{Name: t, Kind: "type"}
	}
	b.compiler.results[filename] = &index.ScalaCompileSpec{
		NotFound: nf,
	}
	return b
}

func (b *importRegistryBuilder) provides(from string, imports ...string) *importRegistryBuilder {
	l, err := label.Parse(from)
	if err != nil {
		b.t.Fatal(err)
	}
	b.registry.Provides(l, imports)
	return b
}

func (b *importRegistryBuilder) depends(deps map[string]string) *importRegistryBuilder {
	for src, dst := range deps {
		b.registry.Depends(src, dst)
	}
	return b
}

func (b *importRegistryBuilder) build() *importRegistry {
	b.registry.OnResolve()
	return b.registry
}

type fakeRuleRegistry struct {
	t     *testing.T
	rules map[label.Label]*index.ScalaRuleSpec
}

func (rr *fakeRuleRegistry) GetScalaRule(from label.Label) (*index.ScalaRuleSpec, bool) {
	rule, ok := rr.rules[from]
	return rule, ok
}

type fakeClassRegistry struct {
	t *testing.T
}

// CrossResolve implements the CrossResolver interface.
func (r *fakeClassRegistry) CrossResolve(c *config.Config, ix *resolve.RuleIndex, imp resolve.ImportSpec, lang string) []resolve.FindResult {
	return nil
}

type fakeCompiler struct {
	t       *testing.T
	results map[string]*index.ScalaCompileSpec
}

func (fc *fakeCompiler) Compile(dir, filename string) (*index.ScalaCompileSpec, error) {
	fc.t.Log("compiler", filename, fc.results)
	spec, ok := fc.results[filename]
	if !ok {
		return nil, fmt.Errorf("file not found: %q", filename)
	}
	return spec, nil
}
