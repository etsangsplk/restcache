package restcache

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"stackmachine.com/blobstore"

	"goji.io"
	"goji.io/pat"
)

type CAS struct {
	store   blobstore.Client
	handler http.Handler
}

func NewServer(store blobstore.Client) *CAS {
	c := CAS{store: store}
	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/cas/:key"), c.Get)
	mux.HandleFunc(pat.Get("/ac/:key"), c.Get)
	mux.HandleFunc(pat.Put("/cas/:key"), c.Put)
	mux.HandleFunc(pat.Put("/ac/:key"), c.Put)
	c.handler = mux
	return &c
}

func (c *CAS) Get(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	if key == "" {
		w.Header().Set("Cache-Control", "private, no-store")
		http.Error(w, fmt.Sprintf("No key provided"), http.StatusBadRequest)
		return
	}
	exists, err := c.store.Contains(key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error talking to blobstore: %s", err), http.StatusInternalServerError)
		return
	}
	if !exists {
		w.Header().Set("Cache-Control", "private, no-store")
		http.Error(w, fmt.Sprintf("Key %s does not exist", key), http.StatusNotFound)
		return
	}
	reader, n, err := c.store.Get(key)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error talking to blobstore: %s", err), http.StatusInternalServerError)
		return
	}

	age := time.Hour * 24 * 365 // One year
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(age.Seconds())))
	if _, err := io.CopyN(w, reader, n); err != nil {
		http.Error(w, fmt.Sprintf("Error writing out response: %s", err), http.StatusInternalServerError)
		return
	}
}

func (c *CAS) Put(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	key := pat.Param(r, "key")
	if key == "" {
		http.Error(w, fmt.Sprintf("No key provided"), http.StatusBadRequest)
		return
	}
	if r.Body == nil {
		http.Error(w, fmt.Sprintf("No body provided"), http.StatusBadRequest)
		return
	}
	err := c.store.Put(key, r.Body, r.ContentLength)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error talking to blobstore: %s", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *CAS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handler.ServeHTTP(w, r)
}
