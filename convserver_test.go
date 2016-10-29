package main

import (
	"os"
	"testing"

	gock "gopkg.in/h2non/gock.v1"
)

func TestConvertPreiviewKey(t *testing.T) {
	actual := convertPreiviewKey("/foo/bar/baz.qux")
	expected := "/foo/bar/baz-preview.pdf"
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestConvertPreiviewKey2(t *testing.T) {
	actual := convertPreiviewKey("/foo/bar/baz")
	expected := "/foo/bar/baz-preview"
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestResponseJSONFromFile(t *testing.T) {
	file, err := os.Open(".gitignore")
	if err != nil {
		t.Errorf("Failed to open test file %v", err)
	}
	actual := string(responseJSONFromFile(file))
	expected := `{"status":"completed","thumbnails":{"preview":{"content_hash":"b0214b0ba0fa51ebf8bd66ba20a82ee9","content_type":"application/pdf","width":500,"height":500}}}`
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func TestRunCommand(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_ACCESS_KEY_ID", "foo")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "bar")
	defer gock.Off()
	gock.New("https://test-bucket.s3.amazonaws.com").
		Get("/foo/bar/baz.pptx").
		Reply(200)

	err := runCommand(requestPayload{
		Bucket:             "test-bucket",
		Key:                "foo/bar/baz.pptx",
		CallbackURL:        "http://internal-foo-test-api.bar.baz/path/to/callback",
		CallbackHTTPMethod: "PUT",
	})
	// if err != nil {
	expected := "RequestError: send request failed\ncaused by: Get https://test-bucket.s3.amazonaws.com/foo/bar/baz.pptx: gock: cannot match any request" // FIXME
	if err.Error() != expected {
		t.Errorf(`Expected "%v" but got "%v"`, expected, err)
	}
}

func TestSendCallback(t *testing.T) {
	gock.New("http://foo-internal-api.bar.baz").
		Patch("/path/to/callback").
		Reply(200)
	err := sendCallback("PATCH", "http://foo-internal-api.bar.baz/path/to/callback", []byte(`{"status":"ok"}`))
	if err != nil {
		t.Errorf("Expected nil but got %v", err)
	}
}

func TestSendCallbackError(t *testing.T) {
	gock.New("http://foo-internal-api.bar.baz").
		Patch("/path/to/callback").
		Reply(400).
		BodyString("Oh")

	err := sendCallback("PATCH", "http://foo-internal-api.bar.baz/path/to/callback", []byte(`{"status":"ok"}`))
	expected := "Error sending callback: 400 Oh"

	if !(err != nil && err.Error() == expected) {
		t.Errorf(`Expected "%v" but got "%v"`, expected, err)
	}
}
