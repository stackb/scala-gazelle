package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
)

// fakeParser records LoadScalaRule and ParseScalaRule calls.
type fakeParser struct {
	loadCalls  []string
	parseCalls []string
}

func (f *fakeParser) LoadScalaRule(from label.Label, rule *sppb.Rule) error {
	f.loadCalls = append(f.loadCalls, from.String())
	return nil
}

func (f *fakeParser) ParseScalaRule(kind string, from label.Label, dir string, srcs ...string) (*sppb.Rule, error) {
	f.parseCalls = append(f.parseCalls, from.String())
	files := make([]*sppb.File, len(srcs))
	for i, src := range srcs {
		files[i] = &sppb.File{Filename: src}
	}
	return &sppb.Rule{
		Label: from.String(),
		Kind:  kind,
		Files: files,
	}, nil
}

// writeSrc writes a fake scala source file and returns its dir.
func writeSrc(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// parseOnce runs a fresh MemoParser over the given srcs and returns the
// memoized rule (carrying the correct Sha256 for those files on disk).
func parseOnce(t *testing.T, from label.Label, dir string, srcs ...string) *sppb.Rule {
	t.Helper()
	parser := NewMemoParser(&fakeParser{})
	if _, err := parser.ParseScalaRule("scala_library", from, dir, srcs...); err != nil {
		t.Fatal(err)
	}
	rules := parser.ScalaRules()
	if len(rules) != 1 {
		t.Fatalf("want 1 memoized rule, got %d", len(rules))
	}
	return rules[0]
}

func TestMemoParserSeedDoesNotLoadSymbols(t *testing.T) {
	dir := t.TempDir()
	writeSrc(t, dir, "a.scala", "class A")
	from := label.New("", "pkg", "a")
	cached := parseOnce(t, from, dir, "a.scala")

	next := &fakeParser{}
	parser := NewMemoParser(next)
	parser.SeedScalaRule(from, cached)

	if len(next.loadCalls) != 0 {
		t.Fatalf("seeding must not load symbols, got LoadScalaRule calls: %v", next.loadCalls)
	}
}

func TestMemoParserHitLoadsSymbolsExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	writeSrc(t, dir, "a.scala", "class A")
	from := label.New("", "pkg", "a")
	cached := parseOnce(t, from, dir, "a.scala")

	next := &fakeParser{}
	parser := NewMemoParser(next)
	parser.SeedScalaRule(from, cached)

	for i := 0; i < 2; i++ {
		got, err := parser.ParseScalaRule("scala_library", from, dir, "a.scala")
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(cached.Label, got.Label); diff != "" {
			t.Fatalf("rule label (-want +got):\n%s", diff)
		}
	}

	if len(next.parseCalls) != 0 {
		t.Fatalf("sha256 match must not re-parse, got ParseScalaRule calls: %v", next.parseCalls)
	}
	if diff := cmp.Diff([]string{"//pkg:a"}, next.loadCalls); diff != "" {
		t.Fatalf("symbols must load exactly once, on first hit (-want +got):\n%s", diff)
	}
}

func TestMemoParserStaleSeedReparses(t *testing.T) {
	dir := t.TempDir()
	writeSrc(t, dir, "a.scala", "class A")
	from := label.New("", "pkg", "a")
	cached := parseOnce(t, from, dir, "a.scala")

	// file content changes after the rule was cached
	writeSrc(t, dir, "a.scala", "class A { def b = 1 }")

	next := &fakeParser{}
	parser := NewMemoParser(next)
	parser.SeedScalaRule(from, cached)

	if _, err := parser.ParseScalaRule("scala_library", from, dir, "a.scala"); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff([]string{"//pkg:a"}, next.parseCalls); diff != "" {
		t.Fatalf("sha256 mismatch must re-parse (-want +got):\n%s", diff)
	}
}

func TestMemoParserScalaRulesOmitsUnusedSeeds(t *testing.T) {
	dir := t.TempDir()
	writeSrc(t, dir, "a.scala", "class A")
	live := label.New("", "pkg", "live")
	deleted := label.New("", "pkg", "deleted")
	cached := parseOnce(t, live, dir, "a.scala")

	next := &fakeParser{}
	parser := NewMemoParser(next)
	parser.SeedScalaRule(live, cached)
	// a rule that was cached on a previous run but no longer exists in the
	// tree: it is seeded but never visited by the walk
	parser.SeedScalaRule(deleted, &sppb.Rule{Label: deleted.String(), Kind: "scala_library"})

	if _, err := parser.ParseScalaRule("scala_library", live, dir, "a.scala"); err != nil {
		t.Fatal(err)
	}

	var got []string
	for _, r := range parser.ScalaRules() {
		got = append(got, r.Label)
	}
	if diff := cmp.Diff([]string{"//pkg:live"}, got); diff != "" {
		t.Fatalf("ScalaRules must omit seeded-but-unused rules (-want +got):\n%s", diff)
	}
}

func TestMemoParserLoadScalaRuleMarksUsed(t *testing.T) {
	from := label.New("", "pkg", "a")
	next := &fakeParser{}
	parser := NewMemoParser(next)

	if err := parser.LoadScalaRule(from, &sppb.Rule{Label: from.String(), Kind: "scala_library"}); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff([]string{"//pkg:a"}, next.loadCalls); diff != "" {
		t.Fatalf("LoadScalaRule must delegate (-want +got):\n%s", diff)
	}
	if len(parser.ScalaRules()) != 1 {
		t.Fatal("explicitly loaded rules must be retained by ScalaRules")
	}
}
