package rediscas

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"goji.io"
	"goji.io/pat"

	"github.com/go-redis/redis"
)

type CAS struct {
	AccessKey string
	SecretKey string

	client  *redis.Client
	handler http.Handler
}

func NewServer(client *redis.Client) *CAS {
	c := CAS{client: client}
	mux := goji.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		// TODO: Use basic authentication once Bazel support lands
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := pat.Param(r, "user")
			pass := pat.Param(r, "pass")
			if user != c.AccessKey && pass != c.SecretKey {
				http.Error(w, "Unauthorized.", 401)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	mux.HandleFunc(pat.Get("/:user/:pass/:key"), c.Get)
	mux.HandleFunc(pat.Put("/:user/:pass/:key"), c.Put)
	c.handler = mux
	return &c
}

func (c *CAS) Get(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	if key == "" {
		http.Error(w, fmt.Sprintf("No key provided"), http.StatusBadRequest)
		return
	}
	exists, err := c.client.Exists(key).Result()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error talking to Redis: %s", err), http.StatusInternalServerError)
		return
	}
	if exists == 0 {
		http.Error(w, fmt.Sprintf("Key %s does not exist", key), http.StatusNotFound)
		return
	}
	val, err := c.client.Get(key).Bytes()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error talking to Redis: %s", err), http.StatusInternalServerError)
		return
	}
	w.Write(val)
}

func (c *CAS) Put(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	if key == "" {
		http.Error(w, fmt.Sprintf("No key provided"), http.StatusBadRequest)
		return
	}
	if r.Body == nil {
		http.Error(w, fmt.Sprintf("No body provided"), http.StatusBadRequest)
		return
	}
	blob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading request body: %s", err), http.StatusInternalServerError)
		return
	}
	err = c.client.Set(key, blob, 0).Err()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error talking to Redis: %s", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *CAS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handler.ServeHTTP(w, r)
}
