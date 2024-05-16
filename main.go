package main

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nfnt/resize"
)

func main() {
	// Initialize MinIO client object.
	endpoint := "localhost:9000"
	accessKeyID := "youraccesskey"
	secretAccessKey := "yoursecretkey"
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Ensure the bucket exists.
	bucketName := "mybucket"
	location := "us-east-1"

	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			fmt.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		fmt.Printf("Successfully created %s\n", bucketName)
	}

	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Resize the image
		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, "Failed to decode image", http.StatusBadRequest)
			return
		}
		resizedImg := resize.Resize(800, 0, img, resize.Lanczos3)

		// Save the resized image to a temporary file
		tempFile, err := os.CreateTemp("", "resized-*.jpg")
		if err != nil {
			http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tempFile.Name())

		err = jpeg.Encode(tempFile, resizedImg, nil)
		if err != nil {
			http.Error(w, "Failed to encode image", http.StatusInternalServerError)
			return
		}
		tempFile.Seek(0, io.SeekStart)

		// Upload the resized image to MinIO
		objectName := "resized-image.jpg"
		_, err = minioClient.PutObject(context.Background(), bucketName, objectName, tempFile, -1, minio.PutObjectOptions{ContentType: "image/jpeg"})
		if err != nil {
			http.Error(w, "Failed to upload to MinIO", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Successfully uploaded resized image to %s/%s\n", bucketName, objectName)
	})

	fmt.Println("Server started at :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
