package restcache

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"stackmachine.com/blobstore"
)

func TestServer(t *testing.T) {
	// generate a unique key every test run
	b := make([]byte, 36)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	srv := NewServer(blobstore.NewMap())
	url := "/cas/" + hex.EncodeToString(b)
	t.Logf("URL: %s", url)

	{
		// Check for an artifact in the cache
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusNotFound {
			t.Logf("resp: %#v", resp)
			t.Logf("body: %#v", string(body))
			t.Fatalf("Expected lookup to fail")
		}

		hdr := w.Header().Get("Cache-Control")
		if hdr != "private, no-store" {
			t.Errorf("Expected GET to be uncached; %s", hdr)
		}
	}

	{
		// Publish an artifact to the cache
		req := httptest.NewRequest("PUT", url, strings.NewReader("content"))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusNoContent {
			t.Logf("resp: %#v", resp)
			t.Logf("body: %#v", string(body))
			t.Fatalf("Expected upload to succeed")
		}

		hdr := w.Header().Get("Cache-Control")
		if hdr != "private, no-store" {
			t.Errorf("Expected PUT to be uncached; %s", hdr)
		}
	}

	{
		// Retrieve the artifact from the cache
		req := httptest.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK || string(body) != "content" {
			t.Logf("resp: %#v", resp)
			t.Logf("body: %#v", string(body))
			t.Fatalf("Expected upload to succeed")
		}

		hdr := w.Header().Get("Cache-Control")
		if hdr != "max-age=31536000" {
			t.Errorf("Expected GET to be cached; %s", hdr)
		}
	}
}
