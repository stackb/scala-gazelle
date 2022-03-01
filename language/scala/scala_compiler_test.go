package scala

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stackb/scala-gazelle/pkg/index"
)

// TestScalaCompileResponse tests translation of an XML response from the
// compiler to a CompileSpec.
func TestScalaCompileResponse(t *testing.T) {
	for name, tc := range map[string]struct {
		dir          string
		filename     string
		mockResponse string
		want         *index.ScalaCompileSpec
	}{
		"ok": {
			filename: "lib/App.scala",
			mockResponse: `
<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<compileResponse>
  <diagnostic line="57" sev="ERROR" source="lib/App.scala">not found: type Greeter</diagnostic>
  <diagnostic line="67" sev="ERROR" source="lib/App.scala">not found: type Greeter</diagnostic>
</compileResponse>
`,
			want: &index.ScalaCompileSpec{
				NotFound: []*index.NotFoundSymbol{{Name: "Greeter", Kind: "type"}},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
				res.WriteHeader(http.StatusOK)
				res.Write([]byte(tc.mockResponse))
			}))
			defer testServer.Close()

			compiler := &scalaCompiler{
				backendRawURL: testServer.URL,
			}

			if err := compiler.initHTTPClient(); err != nil {
				t.Fatal(err)
			}

			got, err := compiler.Compile(tc.dir, tc.filename)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("importRegistry.completions (-want +got):\n%s", diff)
			}
		})
	}
}
