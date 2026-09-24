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
	"encoding/json"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHumaBucket_HexColor covers the column color: written through the v2
// bucket routes with the same permission as any other bucket edit, and read
// back through v1, which the shipped web client still uses for buckets. The
// v1 update is frozen and never writes the color, so it cannot clear it.
// Fixtures as in TestHumaBucket: project 1 (kanban view 4) owns buckets 1-3;
// projects 9 and 10 are shared to testuser1 read-only and with write, their
// kanban views 36 and 40 carry buckets 9 and 10.
func TestHumaBucket_HexColor(t *testing.T) {
	owned := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/1/views/4/buckets",
		idParam:  "bucket",
		t:        t,
	}
	require.NoError(t, owned.ensureEnv())
	readShared := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/9/views/36/buckets",
		idParam:  "bucket",
		t:        t,
		e:        owned.e,
	}
	writeShared := webHandlerTestV2{
		user:     &testuser1,
		basePath: "/api/v2/projects/10/views/40/buckets",
		idParam:  "bucket",
		t:        t,
		e:        owned.e,
	}
	token := humaTokenFor(t, &testuser1)

	t.Run("Create with color", func(t *testing.T) {
		rec, err := owned.testCreateWithUser(nil, nil, `{"title":"Colored","hex_color":"00ff00"}`)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), `"hex_color":"00ff00"`)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        hexColorBucketID(t, rec.Body.Bytes()),
			"hex_color": "00ff00",
		}, false)
	})
	t.Run("Create strips the leading #", func(t *testing.T) {
		rec, err := owned.testCreateWithUser(nil, nil, `{"title":"Hash","hex_color":"#aabbcc"}`)
		require.NoError(t, err)
		assert.Contains(t, rec.Body.String(), `"hex_color":"aabbcc"`)
		assert.NotContains(t, rec.Body.String(), `#aabbcc`)
	})
	t.Run("Create without color", func(t *testing.T) {
		rec, err := owned.testCreateWithUser(nil, nil, `{"title":"Plain"}`)
		require.NoError(t, err)
		assert.Contains(t, rec.Body.String(), `"hex_color":""`)
	})
	t.Run("Create with a too long color is refused", func(t *testing.T) {
		// Same rule as label and project colors: at most 7 characters (a # plus 6 digits).
		_, err := owned.testCreateWithUser(nil, nil, `{"title":"Too long","hex_color":"ff00ff00"}`)
		require.Error(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, getHTTPErrorCode(err))
		db.AssertMissing(t, "buckets", map[string]interface{}{"title": "Too long"})
	})

	t.Run("Update sets the color", func(t *testing.T) {
		rec, err := owned.testUpdateWithUser(nil, map[string]string{"bucket": "1"}, `{"title":"testbucket1","limit":9999999,"position":1,"hex_color":"ff0000"}`)
		require.NoError(t, err)
		assert.Contains(t, rec.Body.String(), `"hex_color":"ff0000"`)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"title":     "testbucket1",
			"hex_color": "ff0000",
		}, false)
	})
	t.Run("Update with a too long color is refused", func(t *testing.T) {
		_, err := owned.testUpdateWithUser(nil, map[string]string{"bucket": "1"}, `{"title":"testbucket1","hex_color":"ff0000ff"}`)
		require.Error(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, getHTTPErrorCode(err))
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"hex_color": "ff0000",
		}, false)
	})
	t.Run("Write share can set the color", func(t *testing.T) {
		rec, err := writeShared.testUpdateWithUser(nil, map[string]string{"bucket": "10"}, `{"title":"Write colored","hex_color":"123abc"}`)
		require.NoError(t, err)
		assert.Contains(t, rec.Body.String(), `"hex_color":"123abc"`)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        10,
			"hex_color": "123abc",
		}, false)
	})
	t.Run("Read share cannot set the color", func(t *testing.T) {
		_, err := readShared.testUpdateWithUser(nil, map[string]string{"bucket": "9"}, `{"title":"Read colored","hex_color":"abcdef"}`)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, getHTTPErrorCode(err))
		db.AssertMissing(t, "buckets", map[string]interface{}{
			"id":        9,
			"hex_color": "abcdef",
		})
	})

	t.Run("v1 bucket list returns the color", func(t *testing.T) {
		rec := humaRequest(t, owned.e, http.MethodGet, "/api/v1/projects/1/views/4/buckets", "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		colors := hexColorsByBucketID(t, rec.Body.Bytes())
		assert.Equal(t, "ff0000", colors[1])
		assert.Contains(t, colors, int64(2))
		assert.Empty(t, colors[2])
	})
	t.Run("v1 kanban tasks returns the color", func(t *testing.T) {
		rec := humaRequest(t, owned.e, http.MethodGet, "/api/v1/projects/1/views/4/tasks", "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		colors := hexColorsByBucketID(t, rec.Body.Bytes())
		assert.Equal(t, "ff0000", colors[1])
	})
	t.Run("v1 update echoing the bucket keeps the color", func(t *testing.T) {
		// The shipped web client renames a column by posting back every field it
		// read, so the color must survive a v1 update that echoes it.
		rec := humaRequest(t, owned.e, http.MethodGet, "/api/v1/projects/1/views/4/buckets", "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		var buckets []map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &buckets))
		var bucket map[string]json.RawMessage
		for _, b := range buckets {
			if string(b["id"]) == "1" {
				bucket = b
			}
		}
		require.NotNil(t, bucket, "bucket 1 missing from %s", rec.Body.String())
		bucket["title"] = json.RawMessage(`"Renamed in v1"`)
		payload, err := json.Marshal(bucket)
		require.NoError(t, err)

		rec = humaRequest(t, owned.e, http.MethodPost, "/api/v1/projects/1/views/4/buckets/1", string(payload), token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"title":     "Renamed in v1",
			"hex_color": "ff0000",
		}, false)
	})
	t.Run("v1 update without the color keeps it", func(t *testing.T) {
		// An old v1 client, or a tab loaded before the color was set, sends no hex_color.
		rec := humaRequest(t, owned.e, http.MethodPost, "/api/v1/projects/1/views/4/buckets/1", `{"title":"Stale v1 rename"}`, token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"hex_color":"ff0000"`)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"title":     "Stale v1 rename",
			"hex_color": "ff0000",
		}, false)
	})
	t.Run("v1 update does not change the color", func(t *testing.T) {
		rec := humaRequest(t, owned.e, http.MethodPost, "/api/v1/projects/1/views/4/buckets/1", `{"title":"Stale v1 rename","hex_color":"00ff00"}`, token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"hex_color":"ff0000"`)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"hex_color": "ff0000",
		}, false)
	})
	t.Run("v1 create with a too long color is refused", func(t *testing.T) {
		rec := humaRequest(t, owned.e, http.MethodPut, "/api/v1/projects/1/views/4/buckets", `{"title":"v1 too long","hex_color":"ff00ff00"}`, token, "")
		require.Equal(t, http.StatusPreconditionFailed, rec.Code, "body: %s", rec.Body.String())
		var errResp ValidationErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp), "body: %s", rec.Body.String())
		assert.Equal(t, models.ErrCodeInvalidData, errResp.Code)
		require.Len(t, errResp.InvalidFields, 1)
		assert.Contains(t, errResp.InvalidFields[0], "does not validate as runelength(0|7)")
		db.AssertMissing(t, "buckets", map[string]interface{}{"title": "v1 too long"})
	})
}

func hexColorBucketID(t *testing.T, body []byte) int64 {
	t.Helper()
	var b struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &b), "body: %s", string(body))
	require.NotZero(t, b.ID)
	return b.ID
}

// hexColorsByBucketID reads a v1 list of buckets (plain array) into id -> hex_color.
func hexColorsByBucketID(t *testing.T, body []byte) map[int64]string {
	t.Helper()
	var buckets []struct {
		ID       int64  `json:"id"`
		HexColor string `json:"hex_color"`
	}
	require.NoError(t, json.Unmarshal(body, &buckets), "body: %s", string(body))
	colors := make(map[int64]string, len(buckets))
	for _, b := range buckets {
		colors[b.ID] = b.HexColor
	}
	return colors
}
