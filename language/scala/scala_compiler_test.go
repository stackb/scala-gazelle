package scala

import (
	"encoding/json"
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

// TestParseScalaFileSpec checks that we can correctly read the JSON.
func TestParseScalaFileSpec(t *testing.T) {
	for name, tc := range map[string]struct {
		dir          string
		filename     string
		mockResponse string
		want         index.ScalaFileSpec
	}{
		"ok": {
			filename: "com/foo/Utils.scala",
			mockResponse: `
{
	"filename": "com/foo/Utils.scala",
	"packages": [
		"com.foo"
	],
	"imports": [
		"com.typesafe.scalalogging.LazyLogging"
	],
	"traits": [
		"com.foo.RationalUtils"
	],
	"objects": [
		"com.foo.RationalUtils"
	],
	"names": [
		"BigDecimal"
	],
	"extends": {
		"object com.foo.RationalUtils": [
			"RationalPriceUtils"
		],
		"object com.foo.RationalPriceUtils": [
			"RationalPriceUtils"
		],
		"trait com.foo.RationalPriceUtils": [
			"RationalUtils"
		],
		"trait com.foo.RationalUtils": [
			"LazyLogging"
		]
	}
}
`,
			want: index.ScalaFileSpec{
				Filename: "com/foo/Utils.scala",
				Imports:  []string{"com.typesafe.scalalogging.LazyLogging"},
				Packages: []string{"com.foo"},
				Objects:  []string{"com.foo.RationalUtils"},
				Traits:   []string{"com.foo.RationalUtils"},
				Names:    []string{"BigDecimal"},
				Extends: map[string][]string{
					"object com.foo.RationalPriceUtils": {"RationalPriceUtils"},
					"object com.foo.RationalUtils":      {"RationalPriceUtils"},
					"trait com.foo.RationalPriceUtils":  {"RationalUtils"},
					"trait com.foo.RationalUtils":       {"LazyLogging"},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var got index.ScalaFileSpec
			if err := json.Unmarshal([]byte(tc.mockResponse), &got); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ReadScalaFileSpec (-want +got):\n%s", diff)
			}
		})
	}
}
