package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"stackmachine.com/blobstore"
	cas "stackmachine.com/cas"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sess := session.Must(session.NewSession())

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatalf("No bucket provided; please set S3_BUCKET")
	}

	size := int64(1000) * 1e+6 // 1GB
	main := blobstore.NewS3(s3.New(sess), bucket)
	cache := blobstore.NewSynchronized(blobstore.LRU(size, blobstore.NewFileSystem("cas")))
	store := blobstore.Prefixed("cas", blobstore.Cached(main, cache))

	srv := cas.NewServer(store)
	srv.AccessKey = os.Getenv("CAS_ACCESS_KEY_ID")
	srv.SecretKey = os.Getenv("CAS_SECRET_ACCESS_KEY")
	http.ListenAndServe("localhost:"+port, srv)
}
