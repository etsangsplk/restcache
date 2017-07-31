package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"goji.io"
	"goji.io/pat"

	"github.com/go-redis/redis"
)

type CAS struct {
	client *redis.Client
}

func NewCAS(client *redis.Client) *CAS {
	return &CAS{client: client}
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

func handler() http.Handler {
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}

	mux := goji.NewMux()
	mux.Use(func(next http.Handler) http.Handler {
		// TODO: Use basic authentication once Bazel support lands
		access := os.Getenv("CAS_ACCESS_KEY_ID")
		secret := os.Getenv("CAS_SECRET_ACCESS_KEY")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := pat.Param(r, "user")
			pass := pat.Param(r, "pass")
			if user != access && secret != pass {
				http.Error(w, "Unauthorized.", 401)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
	cas := NewCAS(redis.NewClient(opt))
	mux.HandleFunc(pat.Get("/:user/:pass/:key"), cas.Get)
	mux.HandleFunc(pat.Put("/:user/:pass/:key"), cas.Put)
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe("localhost:"+port, handler())
}
