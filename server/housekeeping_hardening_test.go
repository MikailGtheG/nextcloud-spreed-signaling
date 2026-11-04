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
	"sync/atomic"
	"testing"
	"time"

	"github.com/strukturag/nextcloud-spreed-signaling/v2/api"
	logtest "github.com/strukturag/nextcloud-spreed-signaling/v2/log/test"
)

// housekeepingMutationSession intentionally modifies the expiry map from Close.
// This mimics ordinary session teardown and ensures housekeeping never iterates
// a map across a callback that can modify that map.
type housekeepingMutationSession struct {
	Session
	hub    *Hub
	id     api.PublicSessionId
	closed atomic.Bool
}

func (s *housekeepingMutationSession) PublicId() api.PublicSessionId {
	return s.id
}

func (s *housekeepingMutationSession) Close() {
	s.hub.mu.Lock()
	delete(s.hub.expiredSessions, s)
	s.hub.mu.Unlock()
	s.closed.Store(true)
}

func TestHousekeepingExpiredSessionsCanMutateMapOnClose(t *testing.T) {
	h := &Hub{logger: logtest.NewLoggerForTest(t), expiredSessions: make(map[Session]time.Time)}
	now := time.Now()
	sessions := make([]*housekeepingMutationSession, 200)
	for i := range sessions {
		s := &housekeepingMutationSession{hub: h}
		sessions[i] = s
		h.expiredSessions[s] = now.Add(-time.Minute)
	}

	h.checkExpiredSessions(now)

	for i, s := range sessions {
		if !s.closed.Load() {
			t.Errorf("expired session %d was not closed", i)
		}
	}
	if got := len(h.expiredSessions); got != 0 {
		t.Fatalf("expected no expired sessions, got %d", got)
	}
}

func TestHousekeepingDoesNotCloseResumedSession(t *testing.T) {
	h := &Hub{logger: logtest.NewLoggerForTest(t), expiredSessions: make(map[Session]time.Time)}
	now := time.Now()
	s := &housekeepingMutationSession{hub: h}
	h.expiredSessions[s] = now.Add(time.Minute)

	h.checkExpiredSessions(now)

	if s.closed.Load() {
		t.Fatal("unexpired session was closed")
	}
	if got := len(h.expiredSessions); got != 1 {
		t.Fatalf("expected 1 unexpired session, got %d", got)
	}
}
