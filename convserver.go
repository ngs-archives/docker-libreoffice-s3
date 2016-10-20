package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type requestPayload struct {
	Bucket             string `json:"bucket"`
	Key                string `json:"key"`
	CallbackURL        string `json:"callback_url"`
	CallbackHTTPMethod string `json:"callback_method,omitempty"`
}

type responsePayload struct {
	Status     string                    `json:"status"`
	Thumbnails thumbnailsResponsePayload `json:"thumbnails"`
}
type thumbnailsResponsePayload struct {
	Preview fileResponsePayload `json:"preview"`
}

type fileResponsePayload struct {
	ContentHash string `json:"content_hash"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "We don't accept "+r.Method+" requests", http.StatusBadRequest)
			return
		}
		var req requestPayload
		var bytes []byte
		r.Body.Read(bytes)
		if err := json.Unmarshal(bytes, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		go runCommand(req)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func convertPreiviewKey(orgKey string) string {
	ext := filepath.Ext(orgKey)
	return strings.TrimSuffix(orgKey, ext) + "-preview" + ext
}

// http://dev.pawelsz.eu/2014/11/google-golang-compute-md5-of-file.html
func computeMd5(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

func responseJSONFromFile(file *os.File) []byte {
	hashBytes, _ := computeMd5(file.Name())
	hash := hex.EncodeToString(hashBytes)
	if hash == "" {
		hash = "0"
	}
	payload := responsePayload{
		Status: "completed",
		Thumbnails: thumbnailsResponsePayload{
			Preview: fileResponsePayload{
				ContentHash: hash,
				Width:       500, // FIX
				Height:      500, // ME
			},
		},
	}
	b, _ := json.Marshal(&payload)
	return b
}

func runCommand(req requestPayload) {
	tmpfile, err := ioutil.TempFile("", req.Key)
	if err != nil {
		log.Fatalf("Error creating tempfile %v", err)
	}

	dl := s3manager.NewDownloader(nil)
	fs, err := os.Create(tmpfile.Name())
	_, err = dl.Download(fs, &s3.GetObjectInput{
		Bucket: &req.Bucket,
		Key:    &req.Key,
	})
	defer os.Remove(tmpfile.Name())

	cmd := exec.Command("lowriter",
		"--invisible",
		"--convert-to",
		"pdf:writer_pdf_Export",
		"--outdir",
		filepath.Dir(tmpfile.Name()),
		tmpfile.Name())

	err = cmd.Start()
	if err != nil {
		log.Fatalf("Error starting: %v", err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatalf("Error starting: %v", err)
	}

	destKey := convertPreiviewKey(req.Key)

	ul := s3manager.NewUploader(nil)
	_, err = ul.Upload(&s3manager.UploadInput{
		Bucket: &req.Bucket,
		Key:    &destKey,
		Body:   tmpfile,
	})

	method := req.CallbackHTTPMethod
	if method == "" {
		method = "POST"
	}
	json := responseJSONFromFile(tmpfile)
	_, err = http.NewRequest(method, req.CallbackURL, bytes.NewBuffer(json))
	if err != nil {
		log.Fatalf("Error sending callback %v", err)
	}
}
