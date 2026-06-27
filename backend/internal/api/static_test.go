package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPAHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":            {Data: []byte("<!doctype html><title>app</title>")},
		"_app/immutable/app.js": {Data: []byte("console.log('hi')")},
	}
	h := SPAHandler(fsys)

	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
		wantCache  string
	}{
		{"/", http.StatusOK, "<!doctype html>", "no-cache"},
		{"/pins", http.StatusOK, "<!doctype html>", "no-cache"},                       // client route → shell
		{"/_app/immutable/app.js", http.StatusOK, "console.log", "public, max-age=31536000, immutable"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.path, rec.Code, tc.wantStatus)
		}
		if body := rec.Body.String(); len(tc.wantBody) > 0 && !contains(body, tc.wantBody) {
			t.Errorf("%s: body %q does not contain %q", tc.path, body, tc.wantBody)
		}
		if tc.wantCache != "" && rec.Header().Get("Cache-Control") != tc.wantCache {
			t.Errorf("%s: cache-control = %q, want %q", tc.path, rec.Header().Get("Cache-Control"), tc.wantCache)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
