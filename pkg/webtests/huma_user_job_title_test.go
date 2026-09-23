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
	"strings"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/modules/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const humaJobTitlePath = "/api/v2/user/settings/job-title"

// Plaintext of api_tokens fixture 1, owned by user1.
const jobTitleAPITokenUser1 = "tk_2eef46f40ebab3304919ab2e7e39993f75f29d2e"

func jobTitleFromBody(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		JobTitle string `json:"job_title"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "body: %s", string(body))
	return resp.JobTitle
}

// listedJobTitle returns the job_title a paginated user list carries for username;
// nil means the field was omitted. It fails the test when the user is not listed.
func listedJobTitle(t *testing.T, body []byte, username string) *string {
	t.Helper()
	var resp struct {
		Items []struct {
			Username string  `json:"username"`
			JobTitle *string `json:"job_title"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &resp), "body: %s", string(body))
	for _, it := range resp.Items {
		if it.Username == username {
			return it.JobTitle
		}
	}
	require.Failf(t, "user not listed", "%s is missing from %s", username, string(body))
	return nil
}

// All subtests share one env and run in order: later ones build on the title set first.
func TestHumaUserJobTitle(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	token1 := humaTokenFor(t, &testuser1)
	token2 := humaTokenFor(t, &testuser2)

	t.Run("Sets the caller's own job title, trimmed", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"  Gestor de tráfego  "}`, token1, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, "Gestor de tráfego", jobTitleFromBody(t, rec.Body.Bytes()))
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": "Gestor de tráfego"}, false)

		show := humaRequest(t, e, http.MethodGet, "/api/v2/user", "", token1, "")
		require.Equal(t, http.StatusOK, show.Code, "body: %s", show.Body.String())
		assert.Equal(t, "Gestor de tráfego", jobTitleFromBody(t, show.Body.Bytes()))
	})

	t.Run("Another user who shares a project sees it next to the name", func(t *testing.T) {
		// Project 3 belongs to user3 and is shared with both user1 and user2.
		rec := humaRequest(t, e, http.MethodGet, "/api/v2/projects/3/users/search", "", token2, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		jobTitle := listedJobTitle(t, rec.Body.Bytes(), "user1")
		require.NotNil(t, jobTitle, "job_title must be in user1's public JSON: %s", rec.Body.String())
		assert.Equal(t, "Gestor de tráfego", *jobTitle)

		search := humaRequest(t, e, http.MethodGet, "/api/v2/users?q=user1", "", token2, "")
		require.Equal(t, http.StatusOK, search.Code, "body: %s", search.Body.String())
		jobTitle = listedJobTitle(t, search.Body.Bytes(), "user1")
		require.NotNil(t, jobTitle, "job_title must be in user1's public JSON: %s", search.Body.String())
		assert.Equal(t, "Gestor de tráfego", *jobTitle)
	})

	t.Run("More than 100 characters is refused", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"`+strings.Repeat("a", 101)+`"}`, token1, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"code":2002`)
		assert.Contains(t, rec.Body.String(), `"location":"body.job_title"`)
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": "Gestor de tráfego"}, false)
	})

	t.Run("100 characters are accepted, counted in characters after trimming", func(t *testing.T) {
		title := strings.Repeat("é", 100)
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"  `+title+`  "}`, token2, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Equal(t, title, jobTitleFromBody(t, rec.Body.Bytes()))
	})

	t.Run("Cannot set another user's job title", func(t *testing.T) {
		// The route has no user to target: user2 writing only ever changes user2.
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"Designer"}`, token2, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		db.AssertExists(t, "users", map[string]interface{}{"id": 2, "job_title": "Designer"}, false)
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": "Gestor de tráfego"}, false)

		// Naming another user in the body is an unknown property, refused before the handler runs.
		for _, body := range []string{`{"id":1,"job_title":"Hacked"}`, `{"user_id":1,"job_title":"Hacked"}`} {
			rec = humaRequest(t, e, http.MethodPut, humaJobTitlePath, body, token2, "")
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		}
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": "Gestor de tráfego"}, false)
		db.AssertExists(t, "users", map[string]interface{}{"id": 2, "job_title": "Designer"}, false)
	})

	t.Run("A body without job_title is refused instead of clearing it", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{}`, token1, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "body: %s", rec.Body.String())
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": "Gestor de tráfego"}, false)
	})

	t.Run("An empty job title clears it", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":""}`, token1, "")
		require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
		assert.Empty(t, jobTitleFromBody(t, rec.Body.Bytes()))
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": ""}, false)

		// Like email, an empty job_title is omitted from the public user.
		list := humaRequest(t, e, http.MethodGet, "/api/v2/projects/3/users/search", "", token2, "")
		require.Equal(t, http.StatusOK, list.Code, "body: %s", list.Body.String())
		assert.Nil(t, listedJobTitle(t, list.Body.Bytes(), "user1"))
	})

	t.Run("Link shares are refused", func(t *testing.T) {
		share := &models.LinkSharing{ID: 1, Hash: "test", ProjectID: 1, Permission: models.PermissionRead}
		token, err := auth.NewLinkShareJWTAuthtoken(share)
		require.NoError(t, err)

		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"Guest"}`, token, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("API tokens cannot use it", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"Robot"}`, jobTitleAPITokenUser1, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
		db.AssertExists(t, "users", map[string]interface{}{"id": 1, "job_title": ""}, false)
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPut, humaJobTitlePath, `{"job_title":"Nobody"}`, "", "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "body: %s", rec.Body.String())
	})
}
