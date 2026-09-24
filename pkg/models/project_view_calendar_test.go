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
	"encoding/json"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func TestProjectViewKind_Calendar(t *testing.T) {
	t.Run("stored values keep their order", func(t *testing.T) {
		// view_kind is an int column: rows written before calendar existed must keep their meaning.
		kinds := []ProjectViewKind{
			ProjectViewKindList,
			ProjectViewKindGantt,
			ProjectViewKindTable,
			ProjectViewKindKanban,
			ProjectViewKindCalendar,
		}
		for stored, kind := range kinds {
			assert.EqualValues(t, stored, kind)
		}
	})
	t.Run("json round trip", func(t *testing.T) {
		out, err := json.Marshal(&ProjectView{Title: "Calendar", ViewKind: ProjectViewKindCalendar})
		require.NoError(t, err)
		assert.Contains(t, string(out), `"view_kind":"calendar"`)

		back := &ProjectView{}
		require.NoError(t, json.Unmarshal(out, back))
		assert.Equal(t, ProjectViewKindCalendar, back.ViewKind)
	})
	t.Run("the other kinds keep their names", func(t *testing.T) {
		names := map[ProjectViewKind]string{
			ProjectViewKindList:   "list",
			ProjectViewKindGantt:  "gantt",
			ProjectViewKindTable:  "table",
			ProjectViewKindKanban: "kanban",
		}
		for kind, name := range names {
			k := kind
			out, err := json.Marshal(&k)
			require.NoError(t, err)
			assert.JSONEq(t, `"`+name+`"`, string(out))

			var back ProjectViewKind
			require.NoError(t, json.Unmarshal(out, &back))
			assert.Equal(t, kind, back)
		}
	})
	t.Run("an unknown kind is still refused", func(t *testing.T) {
		var k ProjectViewKind
		require.Error(t, json.Unmarshal([]byte(`"agenda"`), &k))
	})
	t.Run("the v2 schema accepts calendar", func(t *testing.T) {
		k := ProjectViewKindList
		assert.Contains(t, k.Schema(nil).Enum, "calendar")
	})
}

func TestProjectView_Calendar(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("create behaves like a list view", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		list := &ProjectView{ProjectID: 1, Title: "Another list", ViewKind: ProjectViewKindList}
		require.NoError(t, list.Create(s, u))
		// A bucket mode sent along is dropped: like a list, a calendar has no buckets.
		calendar := &ProjectView{
			ProjectID:               1,
			Title:                   "Calendar",
			ViewKind:                ProjectViewKindCalendar,
			BucketConfigurationMode: BucketConfigurationModeManual,
		}
		require.NoError(t, calendar.Create(s, u))
		require.NoError(t, s.Commit())

		assert.Equal(t, BucketConfigurationModeNone, calendar.BucketConfigurationMode)
		assert.Zero(t, calendar.DefaultBucketID)
		assert.Zero(t, calendar.DoneBucketID)
		db.AssertExists(t, "project_views", map[string]interface{}{
			"id":                        calendar.ID,
			"project_id":                1,
			"view_kind":                 4,
			"bucket_configuration_mode": BucketConfigurationModeNone,
		}, false)
		db.AssertMissing(t, "buckets", map[string]interface{}{"project_view_id": calendar.ID})
		db.AssertMissing(t, "task_buckets", map[string]interface{}{"project_view_id": calendar.ID})

		s2 := db.NewSession()
		defer s2.Close()
		listPositions, err := s2.Where("project_view_id = ?", list.ID).Count(&TaskPosition{})
		require.NoError(t, err)
		calendarPositions, err := s2.Where("project_view_id = ?", calendar.ID).Count(&TaskPosition{})
		require.NoError(t, err)
		assert.Positive(t, calendarPositions)
		assert.Equal(t, listPositions, calendarPositions)
	})

	t.Run("kanban to calendar and back keeps the buckets like a list", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		view := &ProjectView{
			ID:                      4,
			ProjectID:               1,
			Title:                   "Kanban",
			ViewKind:                ProjectViewKindCalendar,
			BucketConfigurationMode: BucketConfigurationModeManual,
		}
		require.NoError(t, view.Update(s, u))
		require.NoError(t, s.Commit())
		assert.Equal(t, BucketConfigurationModeNone, view.BucketConfigurationMode)

		// A view without buckets keeps the stored bucket ids, which is what restores them on the way back.
		db.AssertExists(t, "project_views", map[string]interface{}{
			"id":                        4,
			"view_kind":                 4,
			"bucket_configuration_mode": BucketConfigurationModeNone,
			"default_bucket_id":         1,
			"done_bucket_id":            3,
		}, false)

		s2 := db.NewSession()
		defer s2.Close()
		view.ViewKind = ProjectViewKindKanban
		view.BucketConfigurationMode = BucketConfigurationModeNone
		require.NoError(t, view.Update(s2, u))
		require.NoError(t, s2.Commit())

		db.AssertExists(t, "project_views", map[string]interface{}{
			"id":                        4,
			"view_kind":                 ProjectViewKindKanban,
			"bucket_configuration_mode": BucketConfigurationModeManual,
			"default_bucket_id":         1,
			"done_bucket_id":            3,
		}, false)
		db.AssertCount(t, "buckets", builder.Eq{"project_view_id": 4}, 3)
	})

	t.Run("lists the same tasks as a list view with the same filter", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Tasks 5 and 6 of project 1 are due after this cut-off in the fixtures.
		const dueAfter = "due_date > '2018-11-29T14:00:00+00:00'"
		list := &ProjectView{ProjectID: 1, Title: "Dated list", ViewKind: ProjectViewKindList, Filter: &TaskCollection{Filter: dueAfter}}
		require.NoError(t, list.Create(s, u))
		calendar := &ProjectView{ProjectID: 1, Title: "Dated calendar", ViewKind: ProjectViewKindCalendar, Filter: &TaskCollection{Filter: dueAfter}}
		require.NoError(t, calendar.Create(s, u))

		readIDs := func(viewID int64) []int64 {
			tc := &TaskCollection{ProjectID: 1, ProjectViewID: viewID}
			result, _, _, err := tc.ReadAll(s, u, "", 0, 50)
			require.NoError(t, err)
			tasks, is := result.([]*Task)
			require.True(t, is, "expected flat tasks, got %T", result)
			ids := make([]int64, 0, len(tasks))
			for _, task := range tasks {
				ids = append(ids, task.ID)
			}
			return ids
		}

		listIDs := readIDs(list.ID)
		assert.Contains(t, listIDs, int64(5))
		assert.Contains(t, listIDs, int64(6))
		assert.ElementsMatch(t, listIDs, readIDs(calendar.ID))
	})
}

func TestProjectDuplicate_KeepsCalendarView(t *testing.T) {
	files.InitTestFileFixtures(t)
	db.LoadAndAssertFixtures(t)
	s := db.NewSession()
	defer s.Close()

	u := &user.User{ID: 1}

	calendar := &ProjectView{ProjectID: 1, Title: "Calendar", ViewKind: ProjectViewKindCalendar}
	require.NoError(t, calendar.Create(s, u))
	originalPositions, err := s.Where("project_view_id = ?", calendar.ID).Count(&TaskPosition{})
	require.NoError(t, err)
	require.Positive(t, originalPositions)

	pd := &ProjectDuplicate{ProjectID: 1}
	can, err := pd.CanCreate(s, u)
	require.NoError(t, err)
	require.True(t, can)
	require.NoError(t, pd.Create(s, u))

	duplicated := []*ProjectView{}
	require.NoError(t, s.Where("project_id = ? AND title = ?", pd.Project.ID, "Calendar").Find(&duplicated))
	require.Len(t, duplicated, 1, "the duplicated project must keep its calendar view")
	assert.NotEqual(t, calendar.ID, duplicated[0].ID)
	assert.Equal(t, ProjectViewKindCalendar, duplicated[0].ViewKind)
	assert.Equal(t, BucketConfigurationModeNone, duplicated[0].BucketConfigurationMode)

	duplicatedPositions, err := s.Where("project_view_id = ?", duplicated[0].ID).Count(&TaskPosition{})
	require.NoError(t, err)
	assert.Equal(t, originalPositions, duplicatedPositions)
	require.NoError(t, s.Commit())

	db.AssertExists(t, "project_views", map[string]interface{}{
		"project_id": pd.Project.ID,
		"title":      "Calendar",
		"view_kind":  4,
	}, false)
}

func TestProject_CreateDefaultViewsWithoutCalendar(t *testing.T) {
	u := &user.User{ID: 1}
	defaultKinds := []ProjectViewKind{
		ProjectViewKindList,
		ProjectViewKindGantt,
		ProjectViewKindTable,
		ProjectViewKindKanban,
	}
	storedKinds := func(t *testing.T, projectID int64) []ProjectViewKind {
		s := db.NewSession()
		defer s.Close()
		views := []*ProjectView{}
		require.NoError(t, s.Where("project_id = ?", projectID).OrderBy("position asc").Find(&views))
		kinds := make([]ProjectViewKind, 0, len(views))
		for _, v := range views {
			kinds = append(kinds, v.ViewKind)
		}
		return kinds
	}

	t.Run("new project", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		project := &Project{Title: "No calendar by default"}
		require.NoError(t, project.Create(s, u))
		require.NoError(t, s.Commit())

		assert.Equal(t, defaultKinds, storedKinds(t, project.ID))
		db.AssertMissing(t, "project_views", map[string]interface{}{
			"project_id": project.ID,
			"view_kind":  ProjectViewKindCalendar,
		})
	})
	t.Run("new saved filter", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		sf := &SavedFilter{Title: "No calendar by default", Filters: &TaskCollection{Filter: "done = false"}}
		require.NoError(t, sf.Create(s, u))
		require.NoError(t, s.Commit())

		assert.Equal(t, defaultKinds, storedKinds(t, getProjectIDFromSavedFilterID(sf.ID)))
	})
}
