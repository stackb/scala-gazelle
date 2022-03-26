package scala

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/google/go-cmp/cmp"
)

func TestVizServer(t *testing.T) {
	for name, tc := range map[string]struct {
		args        []string
		useFreePort bool
		registry    *importRegistry
		route       string
		want        string
		wantStatus  int
	}{
		"manual - block": {
			args:       []string{"-graphviz_block_on_resolve", "-graphviz_port", "64963"},
			registry:   newImportRegistryBuilder(t).build(),
			route:      "/",
			wantStatus: 200,
		},
	} {
		t.Run(name, func(t *testing.T) {
			fs := flag.NewFlagSet("", flag.ExitOnError)
			c := config.New()

			viz := newGraphvizServer(tc.registry)
			viz.RegisterFlags(fs, "fix", c)

			if tc.useFreePort {
				port, lis, err := getFreePort()
				if err != nil {
					t.Fatal(err)
				}
				lis.Close()
				viz.port = fmt.Sprintf("%d", port)
				viz.blockOnResolve = true
			}

			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			if err := viz.CheckFlags(fs, c); err != nil {
				t.Fatal(err)
			}

			wr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.route, nil)

			viz.ServeHTTP(wr, req)

			if err := viz.OnResolvePhase(); err != nil {
				t.Fatal(err)
			}

			if wr.Code != http.StatusOK {
				if diff := cmp.Diff(tc.wantStatus, wr.Code); diff != "" {
					t.Fatalf("status (-want +got):\n%s", diff)
				}
			}

			got := wr.Body.String()
			if !strings.Contains(got, tc.want) {
				t.Fatal("want %q, got %q", tc.want, got)
			}
		})
	}
}

func SkipTestVizMux(t *testing.T) {
	for name, tc := range map[string]struct {
		registry   *importRegistry
		route      string
		want       string
		wantStatus int
	}{
		"empty": {
			registry:   newImportRegistryBuilder(t).build(),
			route:      "/foo",
			want:       "404 page not found\n",
			wantStatus: 404,
		},
		"ui": {
			registry:   newImportRegistryBuilder(t).build(),
			route:      "/ui/home",
			want:       "Scala-Gazelle",
			wantStatus: 200,
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

			if false {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("body (-want +got):\n%s", diff)
				}
			} else {
				if !strings.Contains(got, tc.want) {
					t.Fatal("want %q, got %q", tc.want, got)
				}
			}
		})
	}
}

// getFreePort in this case makes the closing of the listener the responsibility
// of the caller to allow for a guarantee that multiple random port allocations
// don't collide.
func getFreePort() (int, *net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, nil, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, nil, err
	}
	return l.Addr().(*net.TCPAddr).Port, l, nil
}
