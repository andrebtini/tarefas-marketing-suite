// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package websocket

import (
	"slices"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"

	"xorm.io/xorm"
)

// PresenceEvent tells a user that someone who shares a project or a team with
// them came online or went offline. It only exists with service.ms.presence on.
const PresenceEvent = "user.presence"

// A page reload closes the last connection and opens a new one a moment later;
// waiting this long before going offline keeps that from flickering.
const presenceGracePeriod = 30 * time.Second

// PresenceChange is the payload of the user.presence event.
type PresenceChange struct {
	UserID int64 `json:"user_id"`
	Online bool  `json:"online"`
}

type pendingOffline struct {
	stop func() bool
}

// presence tracks who is online. A user is online from their first connection
// until the grace period after their last one closed has run out.
type presence struct {
	mu    sync.Mutex
	grace time.Duration
	// Injectable so tests can run out the grace period without waiting for it.
	afterFunc func(time.Duration, func()) (stop func() bool)
	online    map[int64]bool
	offline   map[int64]*pendingOffline
	queue     []PresenceChange
	draining  bool
	deliver   func(PresenceChange)
}

func newPresence(deliver func(PresenceChange)) *presence {
	return &presence{
		grace: presenceGracePeriod,
		afterFunc: func(d time.Duration, f func()) func() bool {
			return time.AfterFunc(d, f).Stop
		},
		online:  make(map[int64]bool),
		offline: make(map[int64]*pendingOffline),
		deliver: deliver,
	}
}

// connected runs under the hub lock when a user opens their first connection.
func (p *presence) connected(userID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pending, has := p.offline[userID]; has {
		// Back within the grace period: nobody ever saw them go offline.
		pending.stop()
		delete(p.offline, userID)
		return
	}
	if p.online[userID] {
		return
	}
	p.online[userID] = true
	p.enqueue(PresenceChange{UserID: userID, Online: true})
}

// disconnected runs under the hub lock when a user's last connection closes.
func (p *presence) disconnected(userID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.online[userID] {
		return
	}
	if earlier, has := p.offline[userID]; has {
		earlier.stop()
	}
	pending := &pendingOffline{}
	pending.stop = p.afterFunc(p.grace, func() { p.expire(userID, pending) })
	p.offline[userID] = pending
}

func (p *presence) expire(userID int64, pending *pendingOffline) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// A reconnect whose Stop came too late has already replaced or removed it.
	if p.offline[userID] != pending {
		return
	}
	delete(p.offline, userID)
	delete(p.online, userID)
	p.enqueue(PresenceChange{UserID: userID, Online: false})
}

// enqueue must be called with p.mu held. A single drainer delivers the changes
// in the order they happened, so an offline followed by a quick reconnect never
// reaches anyone swapped. Delivery runs outside every lock because it queries the
// database and publishes through the hub.
func (p *presence) enqueue(change PresenceChange) {
	p.queue = append(p.queue, change)
	if p.draining {
		return
	}
	p.draining = true
	go p.drain()
}

func (p *presence) drain() {
	for {
		p.mu.Lock()
		if len(p.queue) == 0 {
			p.draining = false
			p.mu.Unlock()
			return
		}
		change := p.queue[0]
		p.queue = p.queue[1:]
		p.mu.Unlock()

		p.deliver(change)
	}
}

// onlineUserIDs includes users within the grace period and leaves out except.
func (p *presence) onlineUserIDs(except int64) []int64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	ids := make([]int64, 0, len(p.online))
	for id := range p.online {
		if id != except {
			ids = append(ids, id)
		}
	}
	return ids
}

// deliverPresence pushes a change to the connected users who share a project or
// a team with its user. Nobody else connected means no query at all.
func (h *Hub) deliverPresence(change PresenceChange) {
	viewers := h.connectedUserIDs(change.UserID)
	if len(viewers) == 0 {
		return
	}

	s := db.NewSession()
	defer s.Close()

	sharing, err := models.GetUserIDsSharingProjectOrTeam(s, change.UserID)
	if err != nil {
		log.Errorf("WebSocket: could not resolve who may see the presence of user %d: %v", change.UserID, err)
		return
	}
	for _, viewerID := range viewers {
		if sharing[viewerID] {
			h.PublishForUser(viewerID, PresenceEvent, change)
		}
	}
}

func (h *Hub) connectedUserIDs(except int64) []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]int64, 0, len(h.connections))
	for id := range h.connections {
		if id != except {
			ids = append(ids, id)
		}
	}
	return ids
}

// OnlineUserIDsVisibleTo returns, sorted, the online users who share a project
// or a team with viewerID, the viewer itself never included. A user whose last
// connection closed less than the grace period ago still counts as online. The
// list is always empty on a hub that does not track presence.
func (h *Hub) OnlineUserIDsVisibleTo(s *xorm.Session, viewerID int64) ([]int64, error) {
	ids := []int64{}
	if h == nil || h.presence == nil {
		return ids, nil
	}

	online := h.presence.onlineUserIDs(viewerID)
	if len(online) == 0 {
		return ids, nil
	}

	sharing, err := models.GetUserIDsSharingProjectOrTeam(s, viewerID)
	if err != nil {
		return nil, err
	}
	for _, id := range online {
		if sharing[id] {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}
