package rediscas

import (
	"crypto/rand"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-redis/redis"
)

func TestServer(t *testing.T) {
	// generate a unique key every test run
	b := make([]byte, 36)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	srv := NewServer(client)
	srv.AccessKey = "foo"
	srv.SecretKey = "bar"

	url := "/foo/bar/" + string(b)

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
	}
}
