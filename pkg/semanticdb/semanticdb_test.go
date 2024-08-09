package semanticdb

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sppb "github.com/stackb/scala-gazelle/build/stack/gazelle/scala/parse"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

var update = flag.Bool("update", false, "update golden files")

func TestReadJarFile(t *testing.T) {
	for name, tc := range map[string]struct {
		filename string
		wantErr  string
	}{
		"degenerate": {
			wantErr: "opening jar file: open : no such file or directory",
		},
		"example jar": {
			filename: "testdata/example.jar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
				dir = filepath.Join(bwd, "pkg/semanticdb")
			}

			got, err := ReadJarFile(tc.filename)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if err != nil {
				return
			}

			for _, docs := range got {
				for _, doc := range docs.Documents {
					goldenFile := filepath.Join(dir, "testdata", tc.filename, "META-INF", "semanticdb", doc.Uri+".json")

					if *update {
						if err := os.MkdirAll(filepath.Dir(goldenFile), os.ModePerm); err != nil {
							t.Fatal(err)
						}
						if err := protobuf.WriteStableJSONFile(goldenFile, doc); err != nil {
							t.Fatal(err)
						}
						log.Println("Wrote golden file:", goldenFile)
						continue
					}

					var want spb.TextDocument
					if err := protobuf.ReadFile(goldenFile, &want); err != nil {
						t.Fatal(err)
					}

					if diff := cmp.Diff(&want, doc,
						cmpopts.IgnoreUnexported(
							spb.TextDocuments{},
							spb.TextDocument{},
							spb.Range{},
							spb.Location{},
							spb.Scope{},
							spb.Type{},
							spb.LambdaType{},
							spb.TypeRef{},
							spb.SingleType{},
							spb.ThisType{},
							spb.SuperType{},
							spb.ConstantType{},
							spb.IntersectionType{},
							spb.UnionType{},
							spb.WithType{},
							spb.StructuralType{},
							spb.AnnotatedType{},
							spb.ExistentialType{},
							spb.UniversalType{},
							spb.ByNameType{},
							spb.RepeatedType{},
							spb.MatchType{},
							spb.Constant{},
							spb.UnitConstant{},
							spb.BooleanConstant{},
							spb.ByteConstant{},
							spb.ShortConstant{},
							spb.CharConstant{},
							spb.IntConstant{},
							spb.LongConstant{},
							spb.FloatConstant{},
							spb.DoubleConstant{},
							spb.StringConstant{},
							spb.NullConstant{},
							spb.Signature{},
							spb.ClassSignature{},
							spb.MethodSignature{},
							spb.TypeSignature{},
							spb.ValueSignature{},
							spb.SymbolInformation{},
							spb.Documentation{},
							spb.Annotation{},
							spb.Access{},
							spb.PrivateAccess{},
							spb.PrivateThisAccess{},
							spb.PrivateWithinAccess{},
							spb.ProtectedAccess{},
							spb.ProtectedThisAccess{},
							spb.ProtectedWithinAccess{},
							spb.PublicAccess{},
							spb.SymbolOccurrence{},
							spb.Diagnostic{},
							spb.Synthetic{},
							spb.Tree{},
							spb.ApplyTree{},
							spb.FunctionTree{},
							spb.IdTree{},
							spb.LiteralTree{},
							spb.MacroExpansionTree{},
							spb.OriginalTree{},
							spb.SelectTree{},
							spb.TypeApplyTree{},
						)); diff != "" {
						t.Errorf("%s (-want +got):\n%s", doc.Uri, diff)
					}
				}
			}
		})
	}
}

func TestToFile(t *testing.T) {
	for name, tc := range map[string]struct {
		filename string
		wantErr  string
	}{
		"degenerate": {
			wantErr: "opening jar file: open : no such file or directory",
		},
		"example jar": {
			filename: "testdata/example.jar",
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if bwd, ok := os.LookupEnv("BUILD_WORKSPACE_DIRECTORY"); ok {
				dir = filepath.Join(bwd, "pkg/semanticdb")
			}

			got, err := ReadJarFile(tc.filename)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if err != nil {
				return
			}

			for _, docs := range got {
				for _, doc := range docs.Documents {
					file, err := ToFile(doc)
					if err != nil {
						t.Fatal(err)
					}
					goldenFile := filepath.Join(dir, "testdata", tc.filename, "META-INF", "semanticdb", doc.Uri+".file.json")

					if *update {
						if err := os.MkdirAll(filepath.Dir(goldenFile), os.ModePerm); err != nil {
							t.Fatal(err)
						}
						if err := protobuf.WriteStableJSONFile(goldenFile, file); err != nil {
							t.Fatal(err)
						}
						log.Println("Wrote golden file:", goldenFile)
						continue
					}

					var want sppb.File
					if err := protobuf.ReadFile(goldenFile, &want); err != nil {
						t.Fatal(err)
					}

					if diff := cmp.Diff(&want, doc,
						cmpopts.IgnoreUnexported(
							sppb.File{},
						)); diff != "" {
						t.Errorf("%s (-want +got):\n%s", doc.Uri, diff)
					}
				}
			}
		})
	}
}
