package scalaparse

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	sppb "github.com/stackb/scala-gazelle/api/scalaparse"
)

func TestNewHttpScalaParseRequest(t *testing.T) {
	for name, tc := range map[string]struct {
		url      string
		in       *sppb.ScalaParseRequest
		want     *http.Request
		wantBody string
	}{
		"prototypical": {
			url: "http://localhost:3000",
			in: &sppb.ScalaParseRequest{
				Label:    "//app:scala",
				Filename: []string{"A.scala", "B.scala"},
			},
			want: &http.Request{
				Method:        "POST",
				URL:           mustParseURL(t, "http://localhost:3000"),
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Header:        http.Header{"Content-Type": {"application/json"}},
				ContentLength: 53,
				Host:          "localhost:3000",
			},
			wantBody: `{"files":["A.scala","B.scala"],"label":"//app:scala"}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := newHttpScalaParseRequest(tc.url, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			body, err := ioutil.ReadAll(got.Body)
			if err != nil {
				t.Fatal(err)
			}
			gotBody := string(body)
			if diff := cmp.Diff(tc.want, got,
				cmpopts.IgnoreUnexported(http.Request{}),
				cmpopts.IgnoreFields(http.Request{}, "GetBody", "Body"),
			); diff != "" {
				t.Errorf("newHttpScalaParseRequest (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantBody, gotBody); diff != "" {
				t.Errorf("newHttpScalaParseRequest body (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewHttpScalaParseRequestError(t *testing.T) {
	for name, tc := range map[string]struct {
		url  string
		in   *sppb.ScalaParseRequest
		want error
	}{
		"missing-url": {
			want: fmt.Errorf("rpc error: code = InvalidArgument desc = request URL is required"),
		},
		"missing-request": {
			url:  "http://localhost:3000",
			want: fmt.Errorf("rpc error: code = InvalidArgument desc = ScalaParseRequest is required"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, got := newHttpScalaParseRequest(tc.url, tc.in)
			if got == nil {
				t.Fatal("error was expected: %v", tc.want)
			}
			if diff := cmp.Diff(tc.want.Error(), got.Error()); diff != "" {
				t.Errorf("newHttpScalaParseRequest error (-want +got):\n%s", diff)
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal("url parse error: %v", err)
	}
	return u
}
