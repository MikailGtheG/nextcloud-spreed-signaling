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
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
 * Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */
package api

import "testing"

func TestFederationURLValidation(t *testing.T) {
	tests := []struct {
		name        string
		signaling   string
		nextcloud   string
		wantInvalid bool
	}{
		{"https", "https://signal.example.net/", "https://cloud.example.net", false},
		{"local-private-deployment", "http://10.2.0.20:8080/", "http://cloud.internal", false},
		{"websocket", "wss://signal.example.net/", "https://cloud.example.net", false},
		{"missing-signaling-host", "https://", "https://cloud.example.net", true},
		{"disallowed-signaling-scheme", "file:///etc/passwd", "https://cloud.example.net", true},
		{"userinfo", "https://name:password@signal.example.net/", "https://cloud.example.net", true},
		{"fragment", "https://signal.example.net/#fragment", "https://cloud.example.net", true},
		{"missing-nextcloud-host", "https://signal.example.net/", "http://", true},
		{"nextcloud-userinfo", "https://signal.example.net/", "https://name@cloud.example.net/", true},
		{"nextcloud-websocket-scheme", "https://signal.example.net/", "wss://cloud.example.net", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &RoomFederationMessage{
				SignalingUrl: tt.signaling,
				NextcloudUrl: tt.nextcloud,
				Token: "test-token",
			}
			err := msg.CheckValid()
			if tt.wantInvalid && err == nil {
				t.Fatal("expected invalid federation URL to be rejected")
			}
			if !tt.wantInvalid && err != nil {
				t.Fatalf("expected valid federation URL, got %v", err)
			}
		})
	}
}
