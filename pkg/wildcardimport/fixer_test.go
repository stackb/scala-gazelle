package wildcardimport

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMakeImportLine(t *testing.T) {
	for name, tc := range map[string]struct {
		importPrefix string
		symbols      []string
		wantErr      string
		want         string
	}{
		"missing import prefix": {
			symbols: []string{},
			wantErr: "importPrefix must not be empty",
			want:    "",
		},
		"no symbols": {
			symbols:      []string{},
			importPrefix: "foo",
			wantErr:      "must have at least one symbol in list",
			want:         "",
		},
		"one symbol": {
			symbols:      []string{"Bar"},
			importPrefix: "foo",
			want:         "import foo.Bar",
		},
		"two symbols": {
			symbols:      []string{"Bar", "Baz"},
			importPrefix: "foo",
			want:         "import foo.{Bar, Baz}",
		},
		"sorts symbol list": {
			symbols:      []string{"B", "A"},
			importPrefix: "foo",
			want:         "import foo.{A, B}",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := makeImportLine(tc.importPrefix, tc.symbols)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if diff := cmp.Diff(tc.wantErr, gotErr); diff != "" {
				t.Errorf("error (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("result (-want +got):\n%s", diff)
			}
		})
	}
}
