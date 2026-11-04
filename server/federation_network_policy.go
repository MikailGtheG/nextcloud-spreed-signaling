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
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"
)

// These ranges must never be contacted by untrusted federation requests
// unless explicitly permitted by the operator. IsGlobalUnicast alone is not
// sufficient to exclude all IANA special-purpose address ranges.
var federationBlockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
}

func isAllowedFederationAddress(ip net.IP, allowPrivate bool) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if allowPrivate {
		return !address.IsUnspecified() && !address.IsMulticast()
	}
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() {
		return false
	}
	for _, prefix := range federationBlockedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

// dialFederationEndpoint resolves and filters target IPs, then connects to
// the validated literal address. The IP is not resolved a second time during
// DialContext, preventing DNS rebinding between validation and connection.
// Gorilla WebSocket retains the original hostname for TLS name verification.
func dialFederationEndpoint(ctx context.Context, network, address string, allowPrivate bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var permitted []net.IP
	for _, result := range ips {
		if isAllowedFederationAddress(result.IP, allowPrivate) {
			permitted = append(permitted, result.IP)
		}
	}
	if len(permitted) == 0 {
		return nil, fmt.Errorf("federation target has no permitted IP addresses")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ip := range permitted {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no federation address could be connected")
	}
	return nil, lastErr
}
