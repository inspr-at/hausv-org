package store

import (
	"net/http"
	"testing"
)

// HAUSV-144: http.DetectContentType returns "text/xml; charset=utf-8" for XML,
// which must still resolve to a supported extension.
func TestXMLContentTypeResolvesToExtension(t *testing.T) {
	xml := []byte(`<?xml version="1.0" encoding="UTF-8"?><Invoice/>`)
	detected := http.DetectContentType(xml)
	if detected == "application/xml" || detected == "text/xml" {
		t.Skipf("Go returned a param-free type (%q); the bug needs the charset param", detected)
	}
	if _, ok := DocumentExtension(detected); !ok {
		t.Fatalf("DocumentExtension(%q) rejected a valid XML upload", detected)
	}
	if _, ok := IssuePhotoExtension("text/xml; charset=utf-8"); !ok {
		t.Fatal("IssuePhotoExtension rejected text/xml with charset param")
	}
}
