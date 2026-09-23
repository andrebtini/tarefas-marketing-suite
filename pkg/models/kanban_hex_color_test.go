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
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucket_HexColor(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("create stores the color without the leading #", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			Title:         "Colored",
			HexColor:      "#aabbcc",
			ProjectID:     1,
			ProjectViewID: 4,
		}
		require.NoError(t, b.Create(s, u))
		require.NoError(t, s.Commit())

		assert.Equal(t, "aabbcc", b.HexColor)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":              b.ID,
			"title":           "Colored",
			"project_view_id": 4,
			"hex_color":       "aabbcc",
		}, false)
	})
	t.Run("update sets the color", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			ID:            1,
			Title:         "testbucket1",
			HexColor:      "ff0000",
			Limit:         9999999,
			Position:      1,
			ProjectViewID: 4,
		}
		require.NoError(t, b.Update(s, u))
		require.NoError(t, s.Commit())

		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"title":     "testbucket1",
			"limit":     9999999,
			"hex_color": "ff0000",
		}, false)
	})
	t.Run("update strips the leading #", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		b := &Bucket{
			ID:            1,
			Title:         "testbucket1",
			HexColor:      "#00ff00",
			ProjectViewID: 4,
		}
		require.NoError(t, b.Update(s, u))
		require.NoError(t, s.Commit())

		assert.Equal(t, "00ff00", b.HexColor)
		db.AssertExists(t, "buckets", map[string]interface{}{
			"id":        1,
			"hex_color": "00ff00",
		}, false)
	})
	t.Run("update without a color clears it", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		colored := &Bucket{ID: 1, Title: "testbucket1", HexColor: "ff0000", ProjectViewID: 4}
		require.NoError(t, colored.Update(s, u))
		cleared := &Bucket{ID: 1, Title: "testbucket1", ProjectViewID: 4}
		require.NoError(t, cleared.Update(s, u))
		require.NoError(t, s.Commit())

		db.AssertMissing(t, "buckets", map[string]interface{}{
			"id":        1,
			"hex_color": "ff0000",
		})
	})
}

func TestProjectDuplicate_KeepsBucketHexColor(t *testing.T) {
	files.InitTestFileFixtures(t)
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	u := &user.User{ID: 1}

	// bucket 1 (testbucket1) sits in project 1's kanban view 4; bucket 2 stays without a color.
	_, err := s.Where("id = ?", 1).Cols("hex_color").Update(&Bucket{HexColor: "e8e0ff"})
	require.NoError(t, err)

	pd := &ProjectDuplicate{ProjectID: 1}
	can, err := pd.CanCreate(s, u)
	require.NoError(t, err)
	assert.True(t, can)
	require.NoError(t, pd.Create(s, u))

	views := []*ProjectView{}
	require.NoError(t, s.Where("project_id = ?", pd.Project.ID).Find(&views))
	viewIDs := make([]int64, 0, len(views))
	for _, v := range views {
		viewIDs = append(viewIDs, v.ID)
	}
	require.NotEmpty(t, viewIDs)

	duplicated := []*Bucket{}
	require.NoError(t, s.In("project_view_id", viewIDs).Find(&duplicated))
	colors := make(map[string]string, len(duplicated))
	for _, b := range duplicated {
		assert.NotEqual(t, int64(1), b.ID, "the duplicate must get new buckets")
		colors[b.Title] = b.HexColor
	}

	require.Contains(t, colors, "testbucket1")
	assert.Equal(t, "e8e0ff", colors["testbucket1"])
	require.Contains(t, colors, "testbucket2")
	assert.Empty(t, colors["testbucket2"])
}
