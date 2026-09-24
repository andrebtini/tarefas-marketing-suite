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
	"code.vikunja.io/api/pkg/user"

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

// Fixtures: user6 owns project 29, which has direct shares for user11 (read),
// user12 (write), user13 (admin) and user1 (admin), and team shares for team 11
// (user8), team 12 (user9) and team 13 (user10). Project 19, owned by user7,
// sits under it with direct shares for user4, user5 and user6 and team shares
// for team 8 (user1), team 9 (user2) and team 10 (user3, admin).
func TestGetUserIDsVisibleTo(t *testing.T) {
	t.Run("team members are listed only where the viewer is a project admin", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user12 writes on project 29 and administers only project 23, its own.
		ids, err := GetUserIDsVisibleTo(s, 12)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{1: true, 4: true, 5: true, 6: true, 7: true, 11: true, 13: true}, ids)

		// user13 administers project 29 and so project 19 below it.
		ids, err = GetUserIDsVisibleTo(s, 13)
		require.NoError(t, err)
		for _, id := range []int64{2, 3, 8, 9, 10} {
			assert.True(t, ids[id], "user13 administers a project shared with the team of user %d", id)
		}
	})

	t.Run("the relation is not symmetric", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user8 reads project 29 through team 11 and sees its direct shares, user12
		// among them, while user12 does not see the members of team 11.
		ids, err := GetUserIDsVisibleTo(s, 8)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{1: true, 4: true, 5: true, 6: true, 7: true, 11: true, 12: true, 13: true}, ids)

		ids, err = GetUserIDsVisibleTo(s, 12)
		require.NoError(t, err)
		assert.False(t, ids[8])

		// Both still share the project, which is what rules out everyone else.
		sharing, err := GetUserIDsSharingProjectOrTeam(s, 12)
		require.NoError(t, err)
		assert.True(t, sharing[8])
		sharing, err = GetUserIDsSharingProjectOrTeam(s, 8)
		require.NoError(t, err)
		assert.True(t, sharing[12])
	})

	t.Run("a grant on a parent project counts, team shares only for its admins", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// user4 only reads project 19: it sees the owners and direct shares of 19
		// and of its parent 29, and none of the teams.
		ids, err := GetUserIDsVisibleTo(s, 4)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{1: true, 5: true, 6: true, 7: true, 11: true, 12: true, 13: true}, ids)

		// user3 administers project 19 through team 10 and never reaches project 29,
		// yet sees the teams shared on 29, as the user list of project 19 does.
		ids, err = GetUserIDsVisibleTo(s, 3)
		require.NoError(t, err)
		for _, id := range []int64{8, 9, 10} {
			assert.True(t, ids[id], "user %d is in a team shared on the parent of project 19", id)
		}
	})

	t.Run("teammates and nothing in common", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		ids, err := GetUserIDsVisibleTo(s, 14)
		require.NoError(t, err)
		assert.Equal(t, map[int64]bool{6: true}, ids)

		ids, err = GetUserIDsVisibleTo(s, 1)
		require.NoError(t, err)
		assert.True(t, ids[2], "user2 is in team 1 with user1")
		assert.False(t, ids[1], "the user itself is left out")

		ids, err = GetUserIDsVisibleTo(s, 16)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("covers the user list of every reachable project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		for _, viewerID := range []int64{1, 3, 4, 6, 8, 12, 13, 14, 15} {
			visible, err := GetUserIDsVisibleTo(s, viewerID)
			require.NoError(t, err)
			sharing, err := GetUserIDsSharingProjectOrTeam(s, viewerID)
			require.NoError(t, err)
			for id := range visible {
				assert.True(t, sharing[id], "user %d sees user %d without sharing anything", viewerID, id)
			}

			access, err := getProjectAccessForUser(s, viewerID)
			require.NoError(t, err)
			require.NotEmpty(t, access.sortedIDs)
			for _, projectID := range access.sortedIDs {
				listed, err := ListUsersFromProject(s, &Project{ID: projectID}, &user.User{ID: viewerID}, "")
				require.NoError(t, err)
				for _, u := range listed {
					if u.ID != viewerID {
						assert.True(t, visible[u.ID], "project %d lists user %d to user %d", projectID, u.ID, viewerID)
					}
				}
			}
		}
	})
}
