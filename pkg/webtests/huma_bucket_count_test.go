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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHumaBucket_WithCount covers with_count on the v2 bucket list: every
// bucket carries the count the kanban board shows for it, and no task.
//
// Fixtures: project 1 (testuser1) has kanban view 4 with buckets 1, 2 and 3;
// bucket 3 is the done bucket and holds task 2, the only done task. Project 2
// belongs to user3 and is not shared with testuser1; view 8 is its kanban.
func TestHumaBucket_WithCount(t *testing.T) {
	h := webHandlerTestV2{user: &testuser1, t: t}
	require.NoError(t, h.ensureEnv())
	tok := humaTokenFor(t, &testuser1)

	get := func(path string) *httptest.ResponseRecorder {
		return humaRequest(t, h.e, http.MethodGet, path, "", tok, "")
	}

	t.Run("without with_count the count stays zero", func(t *testing.T) {
		rec := get("/api/v2/projects/1/views/4/buckets")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, map[int64]int64{1: 0, 2: 0, 3: 0}, bucketCountsFromList(t, rec.Body.Bytes()))
	})
	t.Run("with_count=false is the plain list", func(t *testing.T) {
		rec := get("/api/v2/projects/1/views/4/buckets?with_count=false")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, map[int64]int64{1: 0, 2: 0, 3: 0}, bucketCountsFromList(t, rec.Body.Bytes()))
	})
	t.Run("counts match the kanban board", func(t *testing.T) {
		rec := get("/api/v2/projects/1/views/4/buckets?with_count=true")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.NotContains(t, rec.Body.String(), `"tasks"`)
		assert.Contains(t, rec.Body.String(), `"total":3`)
		counts := bucketCountsFromList(t, rec.Body.Bytes())
		assert.Equal(t, map[int64]int64{1: 11, 2: 3, 3: 4}, counts)

		board := get("/api/v2/projects/1/views/4/buckets/tasks")
		require.Equal(t, http.StatusOK, board.Code, "body: %s", board.Body.String())
		assert.Equal(t, bucketCountsFromBoard(t, board.Body.Bytes()), counts)
	})
	t.Run("no access gives the same error as the plain list", func(t *testing.T) {
		plain := get("/api/v2/projects/2/views/8/buckets")
		require.Equal(t, http.StatusForbidden, plain.Code, "body: %s", plain.Body.String())
		counted := get("/api/v2/projects/2/views/8/buckets?with_count=true")
		assert.Equal(t, plain.Code, counted.Code)
		assert.Equal(t, problemFromBody(t, plain.Body.Bytes()), problemFromBody(t, counted.Body.Bytes()))
	})
	t.Run("view of another project gives the same error as the plain list", func(t *testing.T) {
		// View 8 belongs to project 2, not to project 1.
		plain := get("/api/v2/projects/1/views/8/buckets")
		require.Equal(t, http.StatusNotFound, plain.Code, "body: %s", plain.Body.String())
		counted := get("/api/v2/projects/1/views/8/buckets?with_count=true")
		assert.Equal(t, plain.Code, counted.Code)
		assert.Equal(t, problemFromBody(t, plain.Body.Bytes()), problemFromBody(t, counted.Body.Bytes()))
	})
	t.Run("invalid with_count is rejected", func(t *testing.T) {
		rec := get("/api/v2/projects/1/views/4/buckets?with_count=maybe")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
	})
	// Runs last: it changes view 4's filter for the rest of this test.
	t.Run("view filter applies to the counts", func(t *testing.T) {
		rec := humaRequest(t, h.e, http.MethodPut, "/api/v2/projects/1/views/4",
			`{"title":"Kanban","view_kind":"kanban","position":4,"filter":{"filter":"done = true"},"bucket_configuration_mode":"manual","default_bucket_id":1,"done_bucket_id":3}`,
			tok, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

		counted := get("/api/v2/projects/1/views/4/buckets?with_count=true")
		require.Equal(t, http.StatusOK, counted.Code, "body: %s", counted.Body.String())
		counts := bucketCountsFromList(t, counted.Body.Bytes())
		assert.Equal(t, map[int64]int64{1: 0, 2: 0, 3: 1}, counts)

		board := get("/api/v2/projects/1/views/4/buckets/tasks")
		require.Equal(t, http.StatusOK, board.Code, "body: %s", board.Body.String())
		assert.Equal(t, bucketCountsFromBoard(t, board.Body.Bytes()), counts)
	})
}

// bucketCountsFromList returns each bucket's count from the bucket list body.
func bucketCountsFromList(t *testing.T, body []byte) map[int64]int64 {
	t.Helper()
	var resp struct {
		Items []struct {
			ID    int64 `json:"id"`
			Count int64 `json:"count"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "body: %s", string(body))
	counts := make(map[int64]int64, len(resp.Items))
	for _, it := range resp.Items {
		counts[it.ID] = it.Count
	}
	return counts
}

// bucketCountsFromBoard returns the count the kanban board reports for each
// bucket, checking it against the cards actually on the board.
func bucketCountsFromBoard(t *testing.T, body []byte) map[int64]int64 {
	t.Helper()
	var resp struct {
		Items []struct {
			ID    int64             `json:"id"`
			Count int64             `json:"count"`
			Tasks []json.RawMessage `json:"tasks"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "body: %s", string(body))
	counts := make(map[int64]int64, len(resp.Items))
	for _, it := range resp.Items {
		// The fixtures stay under the default page size, so every card is on the board.
		require.Len(t, it.Tasks, int(it.Count))
		counts[it.ID] = it.Count
	}
	return counts
}

// problemFromBody decodes the RFC 9457 error body so two errors can be compared.
func problemFromBody(t *testing.T, body []byte) v2ProblemJSON {
	t.Helper()
	var p v2ProblemJSON
	require.NoError(t, json.Unmarshal(body, &p), "body: %s", string(body))
	return p
}
