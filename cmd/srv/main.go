package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis"
	"github.com/kyleconroy/cas"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal(err)
	}
	srv := cas.NewServer(redis.NewClient(opt))
	srv.AccessKey = os.Getenv("CAS_ACCESS_KEY_ID")
	srv.SecretKey = os.Getenv("CAS_SECRET_ACCESS_KEY")
	http.ListenAndServe("localhost:"+port, srv)
}
