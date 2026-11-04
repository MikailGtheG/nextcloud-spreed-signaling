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
package server

import (
	"net"
	"net/http"

	"github.com/strukturag/nextcloud-spreed-signaling/v2/container"
)

// The Janus event channel has no application-level authentication in the
// standard WebSocket event sender. Never trust X-Real-IP or forwarding headers
// for authorizing its peer: they can be provided by an untrusted requester.
// The endpoint should also be excluded from public reverse-proxy routing.
func isAllowedJanusEventPeer(r *http.Request, allowed *container.IPList) bool {
	if allowed == nil || len(r.Header.Values("X-Forwarded-For")) != 0 ||
		len(r.Header.Values("X-Real-IP")) != 0 || len(r.Header.Values("Forwarded")) != 0 {
		return false
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && allowed.Contains(ip)
}
