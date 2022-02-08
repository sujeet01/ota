package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	AWS_S3_REGION     = "ap-south-1"
	AWS_S3_BUCKET     = "mytestbucketchipmonk"
	ACCESS_KEY_ID     = "AKIAYEEAI2VEYYNHBQPS"
	SECRET_ACCESS_KEY = "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"
)

// We will be using this client everywhere in our code
var awsS3Client *s3.Client

func main() {
	configS3()
	http.HandleFunc("/upload", handlerUpload)     // Upload: /upload (upload file named "file")
	http.HandleFunc("/download", handlerDownload) // Download: /download?key={key of the object}&filename={name for the new file}
	http.HandleFunc("/list", handlerList)         // List: /list?prefix={prefix}&delimeter={delimeter}
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// configS3 creates the S3 client
func configS3() {

	creds := credentials.NewStaticCredentialsProvider(ACCESS_KEY_ID, SECRET_ACCESS_KEY, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(creds), config.WithRegion(AWS_S3_REGION))
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	awsS3Client = s3.NewFromConfig(cfg)
}

func showError(w http.ResponseWriter, r *http.Request, status int, message string) {
	http.Error(w, message, status)
}

func handlerList(w http.ResponseWriter, r *http.Request) {

	// There aren't really any folders in S3, but we can emulate them by using "/" in the key names
	// of the objects. In case we want to listen the contents of a "folder" in S3, what we really need
	// to do is to list all objects which have a certain prefix.
	prefix := r.URL.Query().Get("prefix")
	delimeter := r.URL.Query().Get("delimeter")

	paginator := s3.NewListObjectsV2Paginator(awsS3Client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(AWS_S3_BUCKET),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimeter),
	})

	w.Header().Set("Content-Type", "text/html")

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			// Error handling goes here
		}
		for _, obj := range page.Contents {
			// Do whatever you need with each object "obj"
			fmt.Fprintf(w, "<li>File %s</li>", *obj.Key)
		}
	}

	return
}

func handlerUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	// Get a file from the form input name "file"
	file, header, err := r.FormFile("file")
	if err != nil {
		showError(w, r, http.StatusInternalServerError, "Something went wrong retrieving the file from the form")
		return
	}
	defer file.Close()

	filename := header.Filename

	uploader := manager.NewUploader(awsS3Client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(AWS_S3_BUCKET),
		Key:    aws.String(filename),
		Body:   file,
	})
	if err != nil {
		// Do your error handling here
		showError(w, r, http.StatusInternalServerError, "Something went wrong uploading the file")
		return
	}

	fmt.Fprintf(w, "Successfully uploaded to %q\n", AWS_S3_BUCKET)
	return

}

func handlerDownload(w http.ResponseWriter, r *http.Request) {

	// We get the name of the file on the URL
	filename := r.URL.Query().Get("filename")

	// We get the name of the file on the URL
	key := r.URL.Query().Get("key")

	fmt.Println("filename:", filename)
	fmt.Println("key:", key)

	// Create the file
	newFile, err := os.Create(filename)
	if err != nil {
		showError(w, r, http.StatusBadRequest, "Something went wrong creating the local file")
	}
	defer newFile.Close()

	downloader := manager.NewDownloader(awsS3Client)
	_, err = downloader.Download(context.TODO(), newFile, &s3.GetObjectInput{
		Bucket: aws.String(AWS_S3_BUCKET),
		Key:    aws.String(key),
	})

	if err != nil {
		showError(w, r, http.StatusBadRequest, "Something went wrong retrieving the file from S3")
		return
	}

	http.ServeFile(w, r, filename)
}
