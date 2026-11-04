/**
 * Standalone signaling server for the Nextcloud Spreed app.
 * Copyright (C) 2026 contributors
 *
 * @license GNU AGPL version 3 or any later version
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
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
