package semanticdb

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stackb/scala-gazelle/pkg/protobuf"
	spb "github.com/stackb/scala-gazelle/scala/meta/semanticdb"
)

var update = flag.Bool("update", false, "update golden files")

func TestReadJarFile(t *testing.T) {
	for name, tc := range map[string]struct {
		filename         string
		wantErr          string
		wantDocumentsLen int
		wantJson         string
	}{
		"degenerate": {
			wantErr: "opening jar file: open : no such file or directory",
		},
		"example jar": {
			filename:         "testdata/example.jar",
			wantDocumentsLen: 58,
			wantJson: `

`,
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
			if diff := cmp.Diff(tc.wantDocumentsLen, len(got)); diff != "" {
				t.Errorf("wantDocumentsLen (-want +got):\n%s", diff)
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

					if diff := cmp.Diff(&want, got,
						cmpopts.IgnoreUnexported(
							spb.TextDocuments{},
							spb.TextDocument{},
						)); diff != "" {
						t.Errorf("%s (-want +got):\n%s", doc.Uri, diff)
					}

				}

			}
		})
	}
}
