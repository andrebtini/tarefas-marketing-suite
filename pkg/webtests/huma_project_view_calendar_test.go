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
	"fmt"
	"net/http"
	"testing"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type calendarTestView struct {
	ID                      int64  `json:"id"`
	ProjectID               int64  `json:"project_id"`
	ViewKind                string `json:"view_kind"`
	BucketConfigurationMode string `json:"bucket_configuration_mode"`
}

func decodeCalendarTestView(t *testing.T, body []byte) calendarTestView {
	t.Helper()
	var v calendarTestView
	require.NoError(t, json.Unmarshal(body, &v), "body: %s", string(body))
	require.NotZero(t, v.ID, "body: %s", string(body))
	return v
}

// TestHumaProjectView_Calendar covers the calendar view kind on the view
// routes. testuser1 owns project 1 (views 1-4 are list, gantt, table and
// kanban) and has a read-only share on project 9.
func TestHumaProjectView_Calendar(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	token := humaTokenFor(t, &testuser1)

	var calendarID int64
	t.Run("v2 create", func(t *testing.T) {
		// A bucket mode sent along is dropped: like a list, a calendar has no buckets.
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/projects/1/views",
			`{"title":"Calendar","view_kind":"calendar","bucket_configuration_mode":"manual"}`, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		v := decodeCalendarTestView(t, rec.Body.Bytes())
		calendarID = v.ID
		assert.Equal(t, "calendar", v.ViewKind)
		assert.Equal(t, "none", v.BucketConfigurationMode)
		assert.Equal(t, int64(1), v.ProjectID)
		db.AssertExists(t, "project_views", map[string]interface{}{
			"id":                        v.ID,
			"project_id":                1,
			"view_kind":                 4,
			"bucket_configuration_mode": 0,
		}, false)
		db.AssertMissing(t, "buckets", map[string]interface{}{"project_view_id": v.ID})
	})
	t.Run("v2 read", func(t *testing.T) {
		require.NotZero(t, calendarID)
		rec := humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/projects/1/views/%d", calendarID), "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, "calendar", decodeCalendarTestView(t, rec.Body.Bytes()).ViewKind)
	})
	t.Run("v2 list", func(t *testing.T) {
		require.NotZero(t, calendarID)
		rec := humaRequest(t, e, http.MethodGet, "/api/v2/projects/1/views", "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		var env struct {
			Items []calendarTestView `json:"items"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
		kinds := make(map[int64]string, len(env.Items))
		for _, v := range env.Items {
			kinds[v.ID] = v.ViewKind
		}
		assert.Equal(t, "calendar", kinds[calendarID])
		assert.Equal(t, "list", kinds[1])
		assert.Equal(t, "kanban", kinds[4])
	})
	t.Run("v2 update turns a list view into a calendar", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/projects/1/views", `{"title":"Was a list","view_kind":"list"}`, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		id := decodeCalendarTestView(t, rec.Body.Bytes()).ID

		rec = humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/projects/1/views/%d", id),
			`{"title":"Now a calendar","view_kind":"calendar"}`, token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, "calendar", decodeCalendarTestView(t, rec.Body.Bytes()).ViewKind)
		db.AssertExists(t, "project_views", map[string]interface{}{
			"id":        id,
			"title":     "Now a calendar",
			"view_kind": 4,
		}, false)
	})
	t.Run("v2 read share cannot create a calendar", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/projects/9/views", `{"title":"Read only calendar","view_kind":"calendar"}`, token, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
		db.AssertMissing(t, "project_views", map[string]interface{}{"title": "Read only calendar"})
	})
	t.Run("v2 unknown kind is still refused", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/projects/1/views", `{"title":"Agenda","view_kind":"agenda"}`, token, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		db.AssertMissing(t, "project_views", map[string]interface{}{"title": "Agenda"})
	})
	t.Run("v1 create and read", func(t *testing.T) {
		// v1 is frozen, but its generic handler binds the same model JSON, so it takes the new kind with no route change.
		rec := humaRequest(t, e, http.MethodPut, "/api/v1/projects/1/views", `{"title":"v1 calendar","view_kind":"calendar"}`, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		v := decodeCalendarTestView(t, rec.Body.Bytes())
		assert.Equal(t, "calendar", v.ViewKind)
		db.AssertExists(t, "project_views", map[string]interface{}{
			"id":        v.ID,
			"view_kind": 4,
		}, false)

		rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v1/projects/1/views/%d", v.ID), "", token, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, "calendar", decodeCalendarTestView(t, rec.Body.Bytes()).ViewKind)
	})
}

// TestHumaProjectView_CalendarTasks proves a calendar view fetches tasks like a
// list view: the same tasks for the same filter, flat, on v2 and on v1.
func TestHumaProjectView_CalendarTasks(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	token := humaTokenFor(t, &testuser1)

	// Tasks 5 and 6 of project 1 are due after this cut-off in the fixtures.
	const viewFilter = `"filter":{"filter":"due_date > '2018-11-29T14:00:00+00:00'"}`
	createView := func(body string) int64 {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/projects/1/views", body, token, "")
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		return decodeCalendarTestView(t, rec.Body.Bytes()).ID
	}
	listID := createView(`{"title":"Dated list","view_kind":"list",` + viewFilter + `}`)
	calendarID := createView(`{"title":"Dated calendar","view_kind":"calendar",` + viewFilter + `}`)

	t.Run("v2", func(t *testing.T) {
		readIDs := func(viewID int64) ([]int64, int64) {
			rec := humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/projects/1/views/%d/tasks", viewID), "", token, "")
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			var env struct {
				Items []struct {
					ID int64 `json:"id"`
				} `json:"items"`
				Total int64 `json:"total"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
			ids := make([]int64, 0, len(env.Items))
			for _, task := range env.Items {
				ids = append(ids, task.ID)
			}
			return ids, env.Total
		}

		listIDs, listTotal := readIDs(listID)
		calendarIDs, calendarTotal := readIDs(calendarID)
		assert.Contains(t, listIDs, int64(5))
		assert.Contains(t, listIDs, int64(6))
		assert.ElementsMatch(t, listIDs, calendarIDs)
		assert.Equal(t, listTotal, calendarTotal)
	})
	t.Run("v1 returns flat tasks, not buckets", func(t *testing.T) {
		readIDs := func(viewID int64) []int64 {
			rec := humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v1/projects/1/views/%d/tasks", viewID), "", token, "")
			require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
			var items []map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
			ids := make([]int64, 0, len(items))
			for _, item := range items {
				// A kanban answer is a list of buckets, which carry project_view_id and no project_id.
				require.Contains(t, item, "project_id", "expected a task, got %v", item)
				require.NotContains(t, item, "project_view_id", "expected a task, got %v", item)
				var id int64
				require.NoError(t, json.Unmarshal(item["id"], &id))
				ids = append(ids, id)
			}
			return ids
		}

		listIDs := readIDs(listID)
		assert.Contains(t, listIDs, int64(5))
		assert.Contains(t, listIDs, int64(6))
		assert.ElementsMatch(t, listIDs, readIDs(calendarID))
	})
}
