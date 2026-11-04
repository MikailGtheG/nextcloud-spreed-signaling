package server

import (
	"bytes"
	"context"

	"github.com/strukturag/nextcloud-spreed-signaling/v2/talk"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A declared Content-Length is not sufficient protection against a malformed
// or externally constructed request body. The read itself must be bounded.
func TestBackendRequestBodyReadLimit(t *testing.T) {
	b := &BackendServer{}
	wasCalled := false
	handler := b.parseRequestBody(func(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ []byte) {
		wasCalled = true
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/room/test", bytes.NewReader(bytes.Repeat([]byte{'x'}, maxBodySize+1)))
	req.ContentLength = 1 // Deliberately inconsistent declared size.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(talk.HeaderBackendSignalingRandom, "random")
	req.Header.Set(talk.HeaderBackendSignalingChecksum, "checksum")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d", rec.Code)
	}
	if wasCalled {
		t.Fatal("oversized request reached backend handler")
	}
}
