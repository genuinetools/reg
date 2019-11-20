package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types"
)

func createClientAndPing(httpCode int, headerName string, headerValue string) (*Registry, func(), error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerName, headerValue)
		w.WriteHeader(httpCode)
	}))

	auth := types.AuthConfig{ServerAddress: ts.URL}
	r, err := New(context.Background(), auth, Opt{Insecure: true})
	return r, ts.Close, err
}

func TestPing(t *testing.T) {
	testcases := []struct {
		httpCode    int
		headerName  string
		headerValue string
		want        error
	}{
		{
			httpCode:    200,
			headerName:  "whatever",
			headerValue: "whatever",
			want:        ErrNoDockerHeader,
		},
		{
			httpCode:    401,
			headerName:  "wrong",
			headerValue: "whatever",
			want:        ErrNoDockerHeader,
		},
		{
			httpCode:    200,
			headerName:  "docker-distribution-api-version",
			headerValue: "registry/2.0",
			want:        nil,
		},
		{
			httpCode:    401,
			headerName:  "Docker-Distribution-API-Version",
			headerValue: "registry/2.1",
			// Many popular servers do allow unauthenticated image pulls, but require authentication to visit the base url.
			// This conforms to Docker Registry v2 API Specification https://docs.docker.com/registry/spec/api/
			// Thus `401 Unauthorized` is as good response as `200 OK`, if only it has the proper Docker header.
			want: nil,
		},
	}
	for _, tc := range testcases {
		r, closeFunc, err := createClientAndPing(tc.httpCode, tc.headerName, tc.headerValue)
		defer closeFunc()
		if err != tc.want {
			t.Fatalf("when creating client and performing ping for (%v, %q, %q), got error %#v but expected %#v", tc.httpCode, tc.headerName, tc.headerValue, err, tc.want)
		}
		if err != nil {
			continue
		}
		err = r.Ping(context.Background())
		if err != tc.want {
			t.Fatalf("when repeating ping for (%v, %q, %q), got error %#v but expected %#v", tc.httpCode, tc.headerName, tc.headerValue, err, tc.want)
		}
	}
}

func TestPingable(t *testing.T) {
	testcases := []struct {
		registry Registry
		expect   bool
	}{
		{
			registry: Registry{URL: "https://registry-1.docker.io"},
			expect:   true,
		},
		{
			registry: Registry{URL: "https://asia.gcr.io"},
			expect:   true,
		},
	}
	for _, testcase := range testcases {
		actual := testcase.registry.Pingable()
		if testcase.expect != actual {
			t.Fatalf("%s pingable: expected (%v), got (%v)", testcase.registry.URL, testcase.expect, actual)
		}
	}
}
