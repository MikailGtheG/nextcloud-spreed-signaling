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
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/strukturag/nextcloud-spreed-signaling/v2/container"
)

func TestFederationEgressPolicy(t *testing.T) {
	cases := []struct {
		ip string
		want bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2606:4700:4700::1111", true},
		{"127.0.0.1", false},
		{"::1", false},
		{"::ffff:127.0.0.1", false},
		{"169.254.169.254", false},
		{"10.0.0.4", false},
		{"192.168.1.2", false},
		{"172.16.0.1", false},
		{"100.64.0.1", false},
		{"192.0.2.25", false},
		{"198.18.0.1", false},
		{"203.0.113.1", false},
		{"2001:db8::1", false},
	}
	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			if got := isAllowedFederationAddress(net.ParseIP(tc.ip), false); got != tc.want {
				t.Fatalf("isAllowedFederationAddress(%s, false) = %v, want %v", tc.ip, got, tc.want)
			}
		})
	}
	if !isAllowedFederationAddress(net.ParseIP("127.0.0.1"), true) {
		t.Fatal("explicitly allowed private federation must support loopback deployments")
	}
}

func TestFederationEgressDNSPinning(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback network unavailable: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second*5)
	defer cancel()
	if conn, err := dialFederationEndpoint(ctx, "tcp", listener.Addr().String(), false); err == nil {
		conn.Close()
		t.Fatal("loopback federation connection unexpectedly allowed")
	}
	conn, err := dialFederationEndpoint(ctx, "tcp", listener.Addr().String(), true)
	if err != nil {
		t.Fatalf("explicitly permitted loopback dial failed: %v", err)
	}
	conn.Close()
}

func TestJanusEventPeerRestrictions(t *testing.T) {
	allowed := container.DefaultAllowedIPs()
	cases := []struct {
		name, peer string
		forwarded bool
		want bool
	}{
		{"loopback", "127.0.0.1:19000", false, true},
		{"ipv6-loopback", "[::1]:19000", false, true},
		{"private-not-on-allowlist", "10.0.0.10:19000", false, false},
		{"public", "203.0.113.1:19000", false, false},
		{"forged-forwarder", "127.0.0.1:19000", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://localhost/spreed", nil)
			r.RemoteAddr = tc.peer
			if tc.forwarded {
				r.Header.Set("X-Forwarded-For", "127.0.0.1")
			}
			if got := isAllowedJanusEventPeer(r, allowed); got != tc.want {
				t.Fatalf("Janus peer authorized = %v, want %v", got, tc.want)
			}
		})
	}
	private, err := container.ParseIPList("10.0.0.10/32")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "http://localhost/spreed", nil)
	r.RemoteAddr = "10.0.0.10:1234"
	if !isAllowedJanusEventPeer(r, private) {
		t.Fatal("explicit private Janus event sender address denied")
	}
}
