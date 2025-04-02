package resolver_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"

	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/resolver"
	"github.com/stackb/scala-gazelle/pkg/testutil"
)

func Example_newScalaScope_String() {
	scope := resolver.NewTrieScope()

	for _, symbol := range []*resolver.Symbol{
		{
			Type:     sppb.ImportType_PACKAGE,
			Name:     "java.lang",
			Provider: "java",
		},
		{
			Type:     sppb.ImportType_CLASS,
			Name:     "java.lang.String",
			Provider: "java",
		},
		{
			Type:     sppb.ImportType_PACKAGE,
			Name:     "scala",
			Provider: "java",
		},
		{
			Type:     sppb.ImportType_CLASS,
			Name:     "scala.Any",
			Provider: "java",
		},
	} {
		if err := scope.PutSymbol(symbol); err != nil {
			log.Fatal(err)
		}
	}

	scala, _ := resolver.NewScalaScope(scope)

	fmt.Println(scala)
	// output:
	//
	// --- layer 0 ---
	// java <nil>
	// └ lang (java.lang<PACKAGE> //:<java>)
	//   └ String (java.lang.String<CLASS> //:<java>)
	// scala (scala<PACKAGE> //:<java>)
	// └ Any (scala.Any<CLASS> //:<java>)
	//
	// --- layer 1 ---
	// Any (scala.Any<CLASS> //:<java>)
	//
	// --- layer 2 ---
	// String (java.lang.String<CLASS> //:<java>)
	//
	// --- layer 3 ---
	// java <nil>
	// └ lang (java.lang<PACKAGE> //:<java>)
	//   └ String (java.lang.String<CLASS> //:<java>)
	// scala (scala<PACKAGE> //:<java>)
	// └ Any (scala.Any<CLASS> //:<java>)
}

func TestScalaScope(t *testing.T) {
	for name, tc := range map[string]struct {
		known   []*resolver.Symbol
		want    map[string]*resolver.Symbol
		wantErr error
	}{
		"missing everything": {
			wantErr: fmt.Errorf("scala.* package not found (scala builtins will not resolve)"),
		},
		"missing java": {
			known: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "scala.Any",
					Provider: "java",
				},
			},
			wantErr: fmt.Errorf("java.lang.* package not found (java builtins will not resolve)"),
		},
		"resolves expected": {
			known: []*resolver.Symbol{
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "java.lang",
					Provider: "java",
				},
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "java.lang.String",
					Provider: "java",
				},
				{
					Type:     sppb.ImportType_PACKAGE,
					Name:     "scala",
					Provider: "java",
				},
				{
					Type:     sppb.ImportType_CLASS,
					Name:     "scala.Any",
					Provider: "java",
				},
			},
			want: map[string]*resolver.Symbol{
				"java.lang.String": {
					Type:     sppb.ImportType_CLASS,
					Name:     "java.lang.String",
					Provider: "java",
				},
				"String": {
					Type:     sppb.ImportType_CLASS,
					Name:     "java.lang.String",
					Provider: "java",
				},
				"Any": {
					Type:     sppb.ImportType_CLASS,
					Name:     "scala.Any",
					Provider: "java",
				},
				"_root_.java.lang.String": {
					Type:     sppb.ImportType_CLASS,
					Name:     "java.lang.String",
					Provider: "java",
				},
				"_root_.scala.Any": {
					Type:     sppb.ImportType_CLASS,
					Name:     "scala.Any",
					Provider: "java",
				},
				"_root_.String": nil,
				"_root_.Any":    nil,
				"Nope":          nil,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			known := resolver.NewTrieScope()
			for _, symbol := range tc.known {
				if err := known.PutSymbol(symbol); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := resolver.NewScalaScope(known)
			if testutil.ExpectError(t, tc.wantErr, err) {
				return
			}
			got := make(map[string]*resolver.Symbol)
			for name := range tc.want {
				sym, _ := scope.GetSymbol(name)
				got[name] = sym
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}
