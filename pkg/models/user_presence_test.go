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
	"testing"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserIDsSharingProjectOrTeam(t *testing.T) {
	t.Run("owners, direct shares, team shares and teammates count", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		ids, err := GetUserIDsSharingProjectOrTeam(s, 1)
		require.NoError(t, err)
		// user3 owns project 3, which is shared with user1 and user2; user2 is also in
		// team 1 with user1; user8 reaches project 29, shared with user1, through team 11.
		for _, id := range []int64{2, 3, 6, 8} {
			assert.True(t, ids[id], "user %d shares something with user1", id)
		}
		assert.False(t, ids[1], "the user itself is left out")
		// user14 only reaches projects 41 and 42, user16 only owns project 37.
		assert.False(t, ids[14])
		assert.False(t, ids[16])
	})

	t.Run("a grant on a parent project counts", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user15 is shared on project 26 only. user1 reaches it through a share on
		// project 27, three levels above it; user6 owns the whole chain.
		ids, err := GetUserIDsSharingProjectOrTeam(s, 15)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{1: true, 6: true}, ids)

		ids, err = GetUserIDsSharingProjectOrTeam(s, 1)
		require.NoError(t, err)
		assert.True(t, ids[15], "the relation holds from the parent side too")
	})

	t.Run("a team share counts from both sides", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user14 reaches project 41, owned by user6, only through team 16.
		ids, err := GetUserIDsSharingProjectOrTeam(s, 14)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{6: true}, ids)

		ids, err = GetUserIDsSharingProjectOrTeam(s, 6)
		require.NoError(t, err)
		assert.True(t, ids[14])
	})

	t.Run("nothing in common means nobody", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		ids, err := GetUserIDsSharingProjectOrTeam(s, 16)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("a team without any project counts", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Neither user16 nor user20 shares a project with anyone.
		team := &Team{Name: "presence team", CreatedByID: 16}
		_, err := s.Insert(team)
		require.NoError(t, err)
		_, err = s.Insert(&TeamMember{TeamID: team.ID, UserID: 16}, &TeamMember{TeamID: team.ID, UserID: 20})
		require.NoError(t, err)

		ids, err := GetUserIDsSharingProjectOrTeam(s, 16)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{20: true}, ids)

		ids, err = GetUserIDsSharingProjectOrTeam(s, 20)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{16: true}, ids)
	})
}
