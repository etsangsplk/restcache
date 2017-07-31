package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"stackmachine.com/blobstore"
	"stackmachine.com/cas"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	}))

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatalf("No bucket provided; please set S3_BUCKET")
	}

	fs, err := blobstore.NewFileSystem("cas")
	if err != nil {
		log.Fatalf("Failed creating FS store: %s", err)
	}

	size := int64(1000) * 1e+6 // 1GB

	main := blobstore.NewS3(s3.New(sess), bucket)
	cache := blobstore.NewSynchronized(blobstore.LRU(size, fs))
	store := blobstore.Prefixed("cas", blobstore.Cached(main, cache))

	srv := cas.NewServer(store)
	srv.AccessKey = os.Getenv("CAS_ACCESS_KEY_ID")
	srv.SecretKey = os.Getenv("CAS_SECRET_ACCESS_KEY")

	log.Printf("Starting CAS server on port %s", port)
	http.ListenAndServe(":"+port, srv)
}
