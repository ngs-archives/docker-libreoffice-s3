package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	bugsnag "github.com/bugsnag/bugsnag-go"
)

var pdfInfoRegexp = regexp.MustCompile("Page size:\\s+(\\d+) x (\\d+) pts")

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
	ContentType string `json:"content_type"`
	ContentSize int    `json:"content_size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

func main() {
	releaseStage := os.Getenv("ENV")
	if releaseStage == "" {
		releaseStage = "development"
	}
	if bugsnagKey := os.Getenv("BUGSNAG_API_KEY"); bugsnagKey != "" {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       bugsnagKey,
			ReleaseStage: releaseStage,
		})
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "We don't accept "+r.Method+" requests", http.StatusBadRequest)
			return
		}
		var req requestPayload
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		go runCommand(req)
		fmt.Fprintf(w, "OK")
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	err := http.ListenAndServe(":"+port, nil)
	bugsnag.Notify(err)
	log.Fatal(err)
}

func convertPreiviewKey(orgKey string) string {
	ext := filepath.Ext(orgKey)
	suffix := ext
	if ext != "" {
		suffix = ".pdf"
	}
	return strings.TrimSuffix(orgKey, ext) + "-preview" + suffix
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

func responseJSONFromFile(file *os.File) ([]byte, error) {
	hashBytes, err := computeMd5(file.Name())
	if err != nil {
		return []byte{}, err
	}
	hash := hex.EncodeToString(hashBytes)
	if hash == "" {
		hash = "0"
	}
	fi, err := file.Stat()
	if err != nil {
		return []byte{}, err
	}
	size := int(fi.Size())
	w, h, err := pdfSize(file.Name())
	if err != nil {
		return []byte{}, err
	}
	payload := responsePayload{
		Status: "completed",
		Thumbnails: thumbnailsResponsePayload{
			Preview: fileResponsePayload{
				ContentType: "application/pdf",
				ContentHash: hash,
				ContentSize: size,
				Width:       w,
				Height:      h,
			},
		},
	}
	b, err := json.Marshal(&payload)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func runWriter(filename string) error {
	cmd := exec.Command("lowriter",
		"--invisible",
		"--convert-to",
		"pdf:writer_pdf_Export",
		"--outdir",
		filepath.Dir(filename),
		filename)

	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func pdfSize(filename string) (int, int, error) {
	bin := os.Getenv("PDF_INFO_PATH")
	if bin == "" {
		bin = "pdfinfo"
	}
	cmd := exec.Command(bin, filename)

	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	m := pdfInfoRegexp.FindAllStringSubmatch(string(out), 1)
	if len(m) == 0 {
		return 0, 0, errors.New("Invalid pdfinfo output")
	}
	line := m[0]
	w, _ := strconv.Atoi(line[1])
	h, _ := strconv.Atoi(line[2])
	return w, h, nil
}

func sendCallback(method string, url string, json []byte) error {
	if method == "" {
		method = "POST"
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	status := res.StatusCode
	if !(status >= 200 && status < 300) {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Error sending callback: %v %v", status, string(body))
	}
	return err
}

func runCommand(req requestPayload) error {
	defer bugsnag.Recover()
	bugsnagMetadata := bugsnag.MetaData{
		"req": {
			"Bucket":             req.Bucket,
			"Key":                req.Key,
			"CallbackURL":        req.CallbackURL,
			"CallbackHTTPMethod": req.CallbackHTTPMethod,
		},
	}
	tmpfile, err := ioutil.TempFile("", strings.Replace(req.Key, "/", "_", -1))
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}

	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}

	sess := session.New()
	dl := s3manager.NewDownloader(sess)
	fs, err := os.Create(tmpfile.Name())
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}
	_, err = dl.Download(fs, &s3.GetObjectInput{
		Bucket: &req.Bucket,
		Key:    &req.Key,
	})
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}
	defer os.Remove(tmpfile.Name())

	err = runWriter(tmpfile.Name())
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}

	pdf, err := os.Open(strings.TrimSuffix(tmpfile.Name(), filepath.Ext(tmpfile.Name())) + ".pdf")
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}
	defer pdf.Close()

	destKey := convertPreiviewKey(req.Key)
	contentType := "application/pdf"

	ul := s3manager.NewUploader(sess)
	_, err = ul.Upload(&s3manager.UploadInput{
		Bucket:      &req.Bucket,
		Key:         &destKey,
		Body:        pdf,
		ContentType: &contentType,
	})

	json, err := responseJSONFromFile(pdf)
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}
	err = sendCallback(req.CallbackHTTPMethod, req.CallbackURL, json)
	if err != nil {
		bugsnag.Notify(err, bugsnagMetadata)
		return err
	}
	return nil
}
