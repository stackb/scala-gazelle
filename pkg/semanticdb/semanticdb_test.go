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

func TestSemanticImports(t *testing.T) {
	for name, tc := range map[string]struct {
		filename string
		wantErr  string
	}{
		"stringlib": {
			filename: "testdata/stringlib.jar.textdocuments.json",
		},
		"euds": {
			filename: "testdata/edus.jar.textdocuments.json",
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

			var docs spb.TextDocuments
			err = protobuf.ReadFile(tc.filename, &docs)
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

			for _, doc := range docs.Documents {
				got := sppb.File{SemanticImports: SemanticImports(doc)}

				goldenFile := filepath.Join(dir, "testdata", tc.filename+".golden", doc.Uri+".file.json")

				if *update {
					if err := os.MkdirAll(filepath.Dir(goldenFile), os.ModePerm); err != nil {
						t.Fatal(err)
					}
					if err := protobuf.WriteStableJSONFile(goldenFile, &got); err != nil {
						t.Fatal(err)
					}
					log.Println("Wrote golden file:", goldenFile)
					continue
				}

				var want sppb.File
				if err := protobuf.ReadFile(goldenFile, &want); err != nil {
					t.Fatal(err)
				}

				if diff := cmp.Diff(&want, &got, cmpopts.IgnoreUnexported(
					sppb.File{},
				)); diff != "" {
					t.Errorf("%s (-want +got):\n%s", doc.Uri, diff)
				}
			}
		})
	}
}
