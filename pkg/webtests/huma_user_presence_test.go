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

package webtests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"
	"code.vikunja.io/api/pkg/routes"
	"code.vikunja.io/api/pkg/user"
	ws "code.vikunja.io/api/pkg/websocket"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const humaPresencePath = "/api/v2/user/presence"

type wsTestMessage struct {
	Event  string          `json:"event"`
	Error  string          `json:"error"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

func wsTestWrite(ctx context.Context, t *testing.T, conn *websocket.Conn, msg map[string]string) {
	t.Helper()
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	require.NoError(t, conn.Write(ctx, websocket.MessageText, data))
}

func wsTestRead(ctx context.Context, t *testing.T, conn *websocket.Conn) wsTestMessage {
	t.Helper()
	_, data, err := conn.Read(ctx)
	require.NoError(t, err)
	var msg wsTestMessage
	require.NoError(t, json.Unmarshal(data, &msg), "message: %s", string(data))
	return msg
}

// wsTestConnect opens a websocket to the running server and authenticates it as u.
func wsTestConnect(ctx context.Context, t *testing.T, serverURL string, u *user.User) *websocket.Conn {
	t.Helper()
	conn, resp, err := websocket.Dial(ctx, serverURL+"/api/v1/ws", nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	t.Cleanup(func() { _ = conn.CloseNow() })

	wsTestWrite(ctx, t, conn, map[string]string{"action": "auth", "token": humaTokenFor(t, u)})
	msg := wsTestRead(ctx, t, conn)
	require.Equal(t, "auth.success", msg.Action, "unexpected message: %+v", msg)
	return conn
}

// wsTestSubscribePresence returns once the server has applied the subscription:
// a connection handles its messages in order, so the refusal of a made-up event
// sent right after it proves the subscription went through first.
func wsTestSubscribePresence(ctx context.Context, t *testing.T, conn *websocket.Conn) {
	t.Helper()
	wsTestWrite(ctx, t, conn, map[string]string{"action": "subscribe", "event": ws.PresenceEvent})
	wsTestWrite(ctx, t, conn, map[string]string{"action": "subscribe", "event": "ms.barrier"})
	msg := wsTestRead(ctx, t, conn)
	require.Equal(t, "invalid_event", msg.Error, "unexpected message: %+v", msg)
	require.Equal(t, "ms.barrier", msg.Event)
}

func wsTestReadPresence(ctx context.Context, t *testing.T, conn *websocket.Conn) ws.PresenceChange {
	t.Helper()
	msg := wsTestRead(ctx, t, conn)
	require.Equal(t, ws.PresenceEvent, msg.Event, "unexpected message: %+v", msg)
	var change ws.PresenceChange
	require.NoError(t, json.Unmarshal(msg.Data, &change), "data: %s", string(msg.Data))
	return change
}

func presenceIDsFor(t *testing.T, e *echo.Echo, u *user.User) []int64 {
	t.Helper()
	rec := humaRequest(t, e, http.MethodGet, humaPresencePath, "", humaTokenFor(t, u), "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"user_ids":[`, "user_ids must always be an array")
	var body struct {
		UserIDs []int64 `json:"user_ids"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "body: %s", rec.Body.String())
	return body.UserIDs
}

// Fixtures: user1 shares project 3 with its owner user3 and projects 19 and 29 with
// their owners user7 and user6. user14's only link to anyone is project 41, which
// it reaches through team 16 and user6 owns. user16 owns project 37 and shares nothing.
func TestHumaUserPresence(t *testing.T) {
	_, err := setupTestEnv()
	require.NoError(t, err)

	config.ServiceMSPresence.Set(true)
	defer config.ServiceMSPresence.Set(false)
	e := routes.NewEcho()
	routes.RegisterRoutes(e)
	ws.InitHub()

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	user3 := &user.User{ID: 3, Username: "user3"}
	user14 := &user.User{ID: 14, Username: "user14"}
	user16 := &user.User{ID: 16, Username: "user16"}

	t.Run("Empty while nobody is connected", func(t *testing.T) {
		assert.Empty(t, presenceIDsFor(t, e, &testuser1))
	})

	viewer14 := wsTestConnect(ctx, t, srv.URL, user14)
	wsTestSubscribePresence(ctx, t, viewer14)
	viewer1 := wsTestConnect(ctx, t, srv.URL, &testuser1)
	wsTestSubscribePresence(ctx, t, viewer1)
	wsTestConnect(ctx, t, srv.URL, user3)
	wsTestConnect(ctx, t, srv.URL, &testuser6)

	t.Run("Pushes a change only to users who share a project or a team", func(t *testing.T) {
		assert.Equal(t, ws.PresenceChange{UserID: 3, Online: true}, wsTestReadPresence(ctx, t, viewer1))
		assert.Equal(t, ws.PresenceChange{UserID: 6, Online: true}, wsTestReadPresence(ctx, t, viewer1))
		// Changes go out in the order they happened, so user6 arriving first proves
		// user14 never got user3, with whom it shares nothing.
		assert.Equal(t, ws.PresenceChange{UserID: 6, Online: true}, wsTestReadPresence(ctx, t, viewer14))
	})

	t.Run("Lists only the online users the caller shares something with", func(t *testing.T) {
		assert.Equal(t, []int64{3, 6}, presenceIDsFor(t, e, &testuser1))
		assert.Equal(t, []int64{1, 6}, presenceIDsFor(t, e, user3))
		assert.Equal(t, []int64{6}, presenceIDsFor(t, e, user14))
		assert.Empty(t, presenceIDsFor(t, e, user16))
	})

	t.Run("Link shares are refused", func(t *testing.T) {
		share := &models.LinkSharing{ID: 1, Hash: "test", ProjectID: 1, Permission: models.PermissionRead}
		token, err := auth.NewLinkShareJWTAuthtoken(share)
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodGet, humaPresencePath, "", token, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"code":1023`)
	})

	t.Run("API tokens cannot use it", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodGet, humaPresencePath, "", jobTitleAPITokenUser1, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())

		// The fixture token only holds tasks permissions, so also prove that no
		// grantable permission covers the route, even all of them together.
		everything := models.APIPermissions{}
		for group, perms := range models.GetAPITokenRoutes() {
			for perm, route := range perms {
				assert.NotEqualf(t, humaPresencePath, route.Path, "%s.%s makes the presence route grantable", group, perm)
				everything[group] = append(everything[group], perm)
			}
		}
		require.NotEmpty(t, everything)
		c := e.NewContext(httptest.NewRequest(http.MethodGet, humaPresencePath, nil), httptest.NewRecorder())
		assert.False(t, models.CanDoAPIRoute(c, &models.APIToken{APIPermissions: everything}))
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodGet, humaPresencePath, "", "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	})
}

func TestHumaUserPresenceDisabled(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	ws.InitHub()

	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	t.Run("The route is not mounted", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodGet, humaPresencePath, "", humaTokenFor(t, &testuser1), "")
		assert.Equal(t, http.StatusNotFound, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("The event cannot be subscribed to", func(t *testing.T) {
		conn := wsTestConnect(ctx, t, srv.URL, &testuser1)
		wsTestWrite(ctx, t, conn, map[string]string{"action": "subscribe", "event": ws.PresenceEvent})
		msg := wsTestRead(ctx, t, conn)
		assert.Equal(t, "invalid_event", msg.Error)
		assert.Equal(t, ws.PresenceEvent, msg.Event)
	})
}
