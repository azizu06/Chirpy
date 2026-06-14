package auth

import (
	"net/http"
	"testing"
)

func TestGetBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer abc123")
	token, err := GetBearerToken(headers)
	if err != nil {
		t.Fatal(err)
	}
	if token != "abc123" {
		t.Fatalf("expected abc123, got %s", token)
	}
}

func TestGetBearerTokenMissingHeader(t *testing.T) {
	headers := http.Header{}
	_, err := GetBearerToken(headers)
	if err == nil {
		t.Fatal("expected missing header to return an error")
	}
}
