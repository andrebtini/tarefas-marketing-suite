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

package models

import (
	"strings"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jobTitleFromDB(t *testing.T, userID int64) string {
	t.Helper()
	s := db.NewSession()
	defer s.Close()
	u, err := user.GetUserByID(s, userID)
	require.NoError(t, err)
	return u.JobTitle
}

func TestUpdateUserJobTitle(t *testing.T) {
	t.Run("stores the trimmed title", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u, err := user.GetUserByID(s, 2)
		require.NoError(t, err)

		require.NoError(t, UpdateUserJobTitle(s, u, "  Gestor de tráfego  "))
		assert.Equal(t, "Gestor de tráfego", u.JobTitle)
		require.NoError(t, s.Commit())

		db.AssertExists(t, "users", map[string]interface{}{
			"id":        2,
			"username":  "user2",
			"job_title": "Gestor de tráfego",
		}, false)
		assert.Equal(t, "Gestor de tráfego", jobTitleFromDB(t, 2))
		assert.Empty(t, jobTitleFromDB(t, 1), "only the given user's row may change")
	})
	t.Run("refuses more than 100 characters and writes nothing", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u, err := user.GetUserByID(s, 2)
		require.NoError(t, err)

		err = UpdateUserJobTitle(s, u, strings.Repeat("a", 101))
		require.Error(t, err)
		var validationErr ValidationHTTPError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, []string{"job_title"}, validationErr.InvalidFields)
		assert.Equal(t, ErrCodeInvalidData, validationErr.Code)
		require.NoError(t, s.Commit())

		assert.Empty(t, jobTitleFromDB(t, 2))
	})
	t.Run("counts characters, not bytes, and trims before counting", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u, err := user.GetUserByID(s, 2)
		require.NoError(t, err)

		title := strings.Repeat("é", 100)
		require.NoError(t, UpdateUserJobTitle(s, u, "   "+title+"   "))
		require.NoError(t, s.Commit())

		assert.Equal(t, title, jobTitleFromDB(t, 2))
	})
	t.Run("an empty title clears it", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		u, err := user.GetUserByID(s, 2)
		require.NoError(t, err)

		require.NoError(t, UpdateUserJobTitle(s, u, "Designer"))
		require.NoError(t, UpdateUserJobTitle(s, u, ""))
		assert.Empty(t, u.JobTitle)
		require.NoError(t, s.Commit())

		db.AssertExists(t, "users", map[string]interface{}{
			"id":        2,
			"job_title": "",
		}, false)
		assert.Empty(t, jobTitleFromDB(t, 2))
	})
	t.Run("a settings save from a copy loaded earlier does not clear it", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		stale, err := user.GetUserWithEmail(s, &user.User{ID: 2})
		require.NoError(t, err)
		current, err := user.GetUserByID(s, 2)
		require.NoError(t, err)
		require.NoError(t, UpdateUserJobTitle(s, current, "Designer"))

		settings := NewUserGeneralSettings(stale)
		settings.Name = "Another Name"
		require.NoError(t, UpdateUserGeneralSettings(s, stale, settings))
		require.NoError(t, s.Commit())

		assert.Equal(t, "Designer", jobTitleFromDB(t, 2))
	})
}

func TestBotUser_Create_IgnoresJobTitle(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	owner, err := user.GetUserByID(s, 1)
	require.NoError(t, err)

	bot := &BotUser{User: user.User{Username: "bot-job-title", JobTitle: "Set by the owner"}}
	require.NoError(t, bot.Create(s, owner))
	assert.Empty(t, bot.JobTitle)
	require.NoError(t, s.Commit())

	assert.Empty(t, jobTitleFromDB(t, bot.ID))
}
