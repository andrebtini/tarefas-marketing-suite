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
	"context"
	"sync"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClock replaces time.AfterFunc so a test can run out the grace period on demand.
type fakeClock struct {
	mu     sync.Mutex
	timers []*fakeTimer
}

type fakeTimer struct {
	after   time.Duration
	run     func()
	stopped bool
	fired   bool
}

func (c *fakeClock) afterFunc(d time.Duration, f func()) func() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	timer := &fakeTimer{after: d, run: f}
	c.timers = append(c.timers, timer)
	return func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		if timer.stopped || timer.fired {
			return false
		}
		timer.stopped = true
		return true
	}
}

// pending returns the delays of the timers that have neither fired nor been stopped.
func (c *fakeClock) pending() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	delays := []time.Duration{}
	for _, timer := range c.timers {
		if !timer.stopped && !timer.fired {
			delays = append(delays, timer.after)
		}
	}
	return delays
}

// fire marks every pending timer as fired and hands back their callbacks without
// running them, the state in which a Stop call comes too late.
func (c *fakeClock) fire() []func() {
	c.mu.Lock()
	defer c.mu.Unlock()
	runs := []func(){}
	for _, timer := range c.timers {
		if !timer.stopped && !timer.fired {
			timer.fired = true
			runs = append(runs, timer.run)
		}
	}
	return runs
}

// elapse runs out the grace period of every pending timer.
func (c *fakeClock) elapse() {
	for _, run := range c.fire() {
		run()
	}
}

type presenceRecorder struct {
	mu      sync.Mutex
	changes []PresenceChange
}

func (r *presenceRecorder) record(change PresenceChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes = append(r.changes, change)
}

func (r *presenceRecorder) recorded() []PresenceChange {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]PresenceChange{}, r.changes...)
}

// newPresenceHub returns a hub tracking presence on a fake clock. Changes go to
// the recorder when there is one, otherwise out to the hub's connections.
func newPresenceHub(clock *fakeClock, recorder *presenceRecorder) *Hub {
	h := NewHub()
	deliver := h.deliverPresence
	if recorder != nil {
		deliver = recorder.record
	}
	h.presence = newPresence(deliver)
	h.presence.afterFunc = clock.afterFunc
	return h
}

// waitPresenceDelivered returns once every change recorded so far has been delivered.
func waitPresenceDelivered(t *testing.T, h *Hub) {
	t.Helper()
	require.Eventually(t, func() bool {
		h.presence.mu.Lock()
		defer h.presence.mu.Unlock()
		return !h.presence.draining && len(h.presence.queue) == 0
	}, 5*time.Second, time.Millisecond)
}

func presenceConn(userID int64, subscribed bool) *Connection {
	subscriptions := map[string]bool{}
	if subscribed {
		subscriptions[PresenceEvent] = true
	}
	return &Connection{
		userID:        userID,
		authenticated: true,
		subscriptions: subscriptions,
		send:          make(chan OutgoingMessage, 16),
	}
}

func receivedPresence(conn *Connection) []OutgoingMessage {
	msgs := []OutgoingMessage{}
	for len(conn.send) > 0 {
		msgs = append(msgs, <-conn.send)
	}
	return msgs
}

func presenceMessage(userID int64, online bool) OutgoingMessage {
	return OutgoingMessage{Event: PresenceEvent, Data: PresenceChange{UserID: userID, Online: online}}
}

func TestPresenceStaysOnlineWhileAnyConnectionIsOpen(t *testing.T) {
	clock := &fakeClock{}
	recorder := &presenceRecorder{}
	h := newPresenceHub(clock, recorder)
	firstTab := presenceConn(1, false)
	secondTab := presenceConn(1, false)

	h.Register(firstTab)
	h.Register(secondTab)
	waitPresenceDelivered(t, h)
	assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}}, recorder.recorded(), "a second tab must not announce the user again")

	h.Unregister(firstTab)
	waitPresenceDelivered(t, h)
	assert.Empty(t, clock.pending(), "closing one of two tabs must not start the offline countdown")
	assert.Equal(t, []int64{1}, h.presence.onlineUserIDs(0))
	assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}}, recorder.recorded())
}

func TestPresenceGoesOfflineOnlyAfterTheGracePeriod(t *testing.T) {
	clock := &fakeClock{}
	recorder := &presenceRecorder{}
	h := newPresenceHub(clock, recorder)
	firstTab := presenceConn(1, false)
	secondTab := presenceConn(1, false)

	h.Register(firstTab)
	h.Register(secondTab)
	h.Unregister(firstTab)
	h.Unregister(secondTab)
	waitPresenceDelivered(t, h)

	assert.Equal(t, []time.Duration{30 * time.Second}, clock.pending())
	assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}}, recorder.recorded(), "still online while the grace period runs")
	assert.Equal(t, []int64{1}, h.presence.onlineUserIDs(0))

	clock.elapse()
	waitPresenceDelivered(t, h)
	assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}, {UserID: 1, Online: false}}, recorder.recorded())
	assert.Empty(t, h.presence.onlineUserIDs(0))

	h.Register(presenceConn(1, false))
	waitPresenceDelivered(t, h)
	assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}, {UserID: 1, Online: false}, {UserID: 1, Online: true}}, recorder.recorded(), "coming back after the grace period is announced again")
}

func TestPresenceReconnectWithinTheGracePeriod(t *testing.T) {
	t.Run("cancels the pending offline", func(t *testing.T) {
		clock := &fakeClock{}
		recorder := &presenceRecorder{}
		h := newPresenceHub(clock, recorder)
		beforeReload := presenceConn(1, false)

		h.Register(beforeReload)
		h.Unregister(beforeReload)
		require.Len(t, clock.pending(), 1)

		h.Register(presenceConn(1, false))
		assert.Empty(t, clock.pending(), "the reconnect must stop the offline timer")

		clock.elapse()
		waitPresenceDelivered(t, h)
		assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}}, recorder.recorded(), "neither an offline nor a second online")
		assert.Equal(t, []int64{1}, h.presence.onlineUserIDs(0))
	})

	t.Run("wins against a timer that already fired", func(t *testing.T) {
		clock := &fakeClock{}
		recorder := &presenceRecorder{}
		h := newPresenceHub(clock, recorder)
		beforeReload := presenceConn(1, false)

		h.Register(beforeReload)
		h.Unregister(beforeReload)
		late := clock.fire()
		require.Len(t, late, 1)

		h.Register(presenceConn(1, false))
		late[0]()
		waitPresenceDelivered(t, h)
		assert.Equal(t, []PresenceChange{{UserID: 1, Online: true}}, recorder.recorded())
		assert.Equal(t, []int64{1}, h.presence.onlineUserIDs(0))
	})
}

func TestPresenceSubscription(t *testing.T) {
	subscribe := IncomingMessage{Action: ActionSubscribe, Event: PresenceEvent}

	t.Run("accepted when presence is on", func(t *testing.T) {
		conn := presenceConn(1, false)
		conn.hub = newPresenceHub(&fakeClock{}, &presenceRecorder{})

		conn.handleMessage(context.Background(), subscribe)
		assert.True(t, conn.IsSubscribed(PresenceEvent))
		assert.Empty(t, conn.send)
	})

	t.Run("refused when presence is off", func(t *testing.T) {
		conn := presenceConn(1, false)
		conn.hub = NewHub()

		conn.handleMessage(context.Background(), subscribe)
		assert.False(t, conn.IsSubscribed(PresenceEvent))
		msg := <-conn.send
		assert.Equal(t, "invalid_event", msg.Error)
		assert.Equal(t, PresenceEvent, msg.Event)
	})
}

func TestPresenceDisabled(t *testing.T) {
	t.Run("InitHub tracks presence only with service.ms.presence on", func(t *testing.T) {
		InitHub()
		assert.Nil(t, GetHub().presence)

		config.ServiceMSPresence.Set(true)
		defer config.ServiceMSPresence.Set(false)
		InitHub()
		assert.NotNil(t, GetHub().presence)
	})

	t.Run("a hub without presence publishes and lists nothing", func(t *testing.T) {
		h := NewHub()
		viewer := presenceConn(1, true)
		target := presenceConn(3, false)

		h.Register(viewer)
		h.Register(target)
		h.Unregister(target)
		assert.Empty(t, viewer.send)

		ids, err := h.OnlineUserIDsVisibleTo(nil, 1)
		require.NoError(t, err)
		assert.Empty(t, ids)

		var uninitialized *Hub
		ids, err = uninitialized.OnlineUserIDsVisibleTo(nil, 1)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})
}

// Fixtures: user1 shares project 3 with its owner user3 and with user2, who is
// also in team 1 with user1. user16 owns project 37 alone and shares nothing.
func TestPresenceReachesOnlyUsersWhoShareSomething(t *testing.T) {
	s := setupNotificationListenerTest(t)
	clock := &fakeClock{}
	h := newPresenceHub(clock, nil)

	sharing := presenceConn(1, true)
	stranger := presenceConn(16, true)
	unsubscribed := presenceConn(2, false)
	h.Register(sharing)
	h.Register(stranger)
	h.Register(unsubscribed)
	waitPresenceDelivered(t, h)
	assert.Equal(t, []OutgoingMessage{presenceMessage(2, true)}, receivedPresence(sharing))
	assert.Empty(t, receivedPresence(stranger), "user16 must not learn that user1 or user2 came online")
	assert.Empty(t, receivedPresence(unsubscribed))

	target := presenceConn(3, false)
	h.Register(target)
	waitPresenceDelivered(t, h)
	assert.Equal(t, []OutgoingMessage{presenceMessage(3, true)}, receivedPresence(sharing))
	assert.Empty(t, receivedPresence(stranger))
	assert.Empty(t, receivedPresence(unsubscribed), "sharing is not enough without a subscription")

	ids, err := h.OnlineUserIDsVisibleTo(s, 1)
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 3}, ids)
	ids, err = h.OnlineUserIDsVisibleTo(s, 3)
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, ids)
	ids, err = h.OnlineUserIDsVisibleTo(s, 16)
	require.NoError(t, err)
	assert.Empty(t, ids)

	h.Unregister(target)
	waitPresenceDelivered(t, h)
	assert.Empty(t, receivedPresence(sharing), "nothing before the grace period ran out")
	ids, err = h.OnlineUserIDsVisibleTo(s, 1)
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 3}, ids, "still listed within the grace period")

	clock.elapse()
	waitPresenceDelivered(t, h)
	assert.Equal(t, []OutgoingMessage{presenceMessage(3, false)}, receivedPresence(sharing))
	assert.Empty(t, receivedPresence(stranger))
	ids, err = h.OnlineUserIDsVisibleTo(s, 1)
	require.NoError(t, err)
	assert.Equal(t, []int64{2}, ids)
}

// Fixtures: on project 29 user12 has a direct write share, user13 a direct admin
// share, and user8 comes in through team 11. The project's user list shows team
// members only to its admins, and presence follows it.
func TestPresenceFollowsTheProjectUserList(t *testing.T) {
	s := setupNotificationListenerTest(t)
	clock := &fakeClock{}
	h := newPresenceHub(clock, nil)

	writer := presenceConn(12, true)
	admin := presenceConn(13, true)
	h.Register(writer)
	waitPresenceDelivered(t, h)
	h.Register(admin)
	waitPresenceDelivered(t, h)
	assert.Equal(t, []OutgoingMessage{presenceMessage(13, true)}, receivedPresence(writer))

	teamMember := presenceConn(8, true)
	h.Register(teamMember)
	waitPresenceDelivered(t, h)
	assert.Equal(t, []OutgoingMessage{presenceMessage(8, true)}, receivedPresence(admin))
	assert.Empty(t, receivedPresence(writer), "user12 does not administer project 29, so team 11 stays hidden from it")

	ids, err := h.OnlineUserIDsVisibleTo(s, 12)
	require.NoError(t, err)
	assert.Equal(t, []int64{13}, ids)
	ids, err = h.OnlineUserIDsVisibleTo(s, 13)
	require.NoError(t, err)
	assert.Equal(t, []int64{8, 12}, ids)
	ids, err = h.OnlineUserIDsVisibleTo(s, 8)
	require.NoError(t, err)
	assert.Equal(t, []int64{12, 13}, ids, "the team member still sees the direct shares")

	h.Unregister(teamMember)
	clock.elapse()
	waitPresenceDelivered(t, h)
	assert.Equal(t, []OutgoingMessage{presenceMessage(8, false)}, receivedPresence(admin))
	assert.Empty(t, receivedPresence(writer))
}
