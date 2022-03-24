package scala

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestVizUI(t *testing.T) {
	for name, tc := range map[string]struct {
		registry   *importRegistry
		route      string
		want       string
		wantStatus int
	}{
		"empty": {
			registry:   newImportRegistryBuilder(t).build(),
			route:      "/",
			want:       "404 page not found\n",
			wantStatus: 404,
		},
	} {
		t.Run(name, func(t *testing.T) {
			env := &Env{Registry: tc.registry}
			mux := newServeMux(env)
			wr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.route, nil)

			mux.ServeHTTP(wr, req)

			if wr.Code != http.StatusOK {
				if diff := cmp.Diff(tc.wantStatus, wr.Code); diff != "" {
					t.Fatalf("status (-want +got):\n%s", diff)
				}
			}

			got := wr.Body.String()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("body (-want +got):\n%s", diff)
			}
		})
	}
}
