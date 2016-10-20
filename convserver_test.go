package main

import (
	"os"
	"testing"
)

func TestConvertPreiviewKey(t *testing.T) {
	actual := convertPreiviewKey("/foo/bar/baz.qux")
	expected := "/foo/bar/baz-preview.qux"
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
	expected := `{"status":"completed","thumbnails":{"preview":{"content_hash":"b0214b0ba0fa51ebf8bd66ba20a82ee9","width":500,"height":500}}}`
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}
