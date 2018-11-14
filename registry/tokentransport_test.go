package registry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
)

func TestErrBasicAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("www-authenticate", `Basic realm="Registry Realm",service="Docker registry"`)
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	authConfig := types.AuthConfig{
		Username:      "j3ss",
		Password:      "ss3j",
		ServerAddress: ts.URL,
	}
	r, err := New(authConfig.ServerAddress, authConfig, Opt{Insecure: true, Debug: true})
	if err != nil {
		t.Fatalf("expected no error creating client, got %v", err)
	}
	token, err := r.Token(ts.URL)
	if err != ErrBasicAuth {
		t.Fatalf("expected ErrBasicAuth getting token, got %v", err)
	}
	if token != "" {
		t.Fatalf("expected empty token, got %v", err)
	}
}

var authURI string

func oauthFlow(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/oauth2/accesstoken") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token":"abcdef1234"}`))
		return
	}
	if strings.HasPrefix(r.URL.Path, "/oauth2/token") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"abcdef1234"}`))
		return
	}
	auth := r.Header.Get("authorization")
	if !strings.HasPrefix(auth, "Bearer") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if authURI != "" {
			w.Header().Set("www-authenticate", `Bearer realm="`+authURI+`/oauth2/token",service="my.endpoint.here"`)
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED","message":"authentication required","detail":null}]}`))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func TestBothTokenAndAccessTokenWork(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(oauthFlow))
	defer ts.Close()

	for _, which := range []string{"token", "accesstoken"} {
		authURI = ts.URL + "/oauth2/" + which + "?service=my.endpoint.here"
		authConfig := types.AuthConfig{
			Username:      "abc",
			Password:      "123",
			ServerAddress: ts.URL,
		}
		authConfig.Email = "me@email.com"
		r, err := New(ts.URL, authConfig, Opt{Insecure: true, Debug: true})
		if err != nil {
			t.Fatalf("expected no error creating client, got %v", err)
		}
		token, err := r.Token(ts.URL)
		if err != nil {
			t.Fatalf("err getting token from url: %v err: %v", ts.URL, err)
		}
		if token == "" {
			t.Fatalf("error got empty token")
		}
	}
}
