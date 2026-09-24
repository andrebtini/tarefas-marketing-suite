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
	"net/http"
	"net/http/httptest"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/license"
	"code.vikunja.io/api/pkg/user"
	"code.vikunja.io/api/pkg/web"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// listBucketCounts reads the bucket list the way the v2 route does and returns
// each bucket's count by bucket id.
func listBucketCounts(t *testing.T, s *xorm.Session, a web.Auth, projectID, viewID int64, withCount bool) map[int64]int64 {
	t.Helper()
	b := &Bucket{ProjectID: projectID, ProjectViewID: viewID}
	if withCount {
		b.SetWithCount()
	}
	result, _, _, err := b.ReadAll(s, a, "", 0, 0)
	require.NoError(t, err)
	buckets, is := result.([]*Bucket)
	require.True(t, is, "ReadAll returned %T", result)
	counts := make(map[int64]int64, len(buckets))
	for _, bb := range buckets {
		assert.Empty(t, bb.Tasks, "the bucket list must not carry tasks")
		counts[bb.ID] = bb.Count
	}
	return counts
}

// kanbanBucketCounts returns the count the kanban board itself reports for
// each bucket of the view.
func kanbanBucketCounts(t *testing.T, s *xorm.Session, a web.Auth, projectID, viewID int64) map[int64]int64 {
	t.Helper()
	tc := &TaskCollection{ProjectID: projectID, ProjectViewID: viewID}
	result, _, _, err := tc.ReadAll(s, a, "", 0, 0)
	require.NoError(t, err)
	board, is := result.([]*Bucket)
	require.True(t, is, "a manual kanban view must return buckets, got %T", result)
	counts := make(map[int64]int64, len(board))
	for _, bb := range board {
		// Page 0 loads every task, so the count must equal the cards on the board.
		require.Len(t, bb.Tasks, int(bb.Count))
		counts[bb.ID] = bb.Count
	}
	return counts
}

// createCountSavedFilter creates a saved filter of the given owner matching
// tasks 1, 2 and 3 of project 1, and returns its pseudo project id and its
// kanban view. Creating it puts the three tasks in the view's first bucket.
func createCountSavedFilter(t *testing.T, s *xorm.Session, owner *user.User) (int64, *ProjectView) {
	t.Helper()
	sf := &SavedFilter{
		Title:   "bucket counts",
		Filters: &TaskCollection{Filter: "id in 1,2,3"},
	}
	require.NoError(t, sf.Create(s, owner))

	projectID := getProjectIDFromSavedFilterID(sf.ID)
	view := &ProjectView{}
	exists, err := s.Where("project_id = ? AND view_kind = ?", projectID, ProjectViewKindKanban).Get(view)
	require.NoError(t, err)
	require.True(t, exists)
	return projectID, view
}

// Fixtures: project 1 (user 1) has kanban view 4 with buckets 1, 2 and 3. Bucket
// 3 is the view's done bucket and holds task 2, the only done task. Task 51 of
// project 1 is soft-deleted and sits in no bucket. Project 9 is shared read-only
// with user 1, and its kanban view 36 has bucket 9 (task 18) and bucket 25
// (empty). Project 2 belongs to user 3 and is not shared with user 1.
func TestBucket_ReadAllWithCount(t *testing.T) {
	u := &user.User{ID: 1}

	t.Run("without with_count the count stays zero", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		assert.Equal(t, map[int64]int64{1: 0, 2: 0, 3: 0}, listBucketCounts(t, s, u, 1, 4, false))
	})
	t.Run("counts match the kanban board", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		counts := listBucketCounts(t, s, u, 1, 4, true)
		assert.Equal(t, map[int64]int64{1: 11, 2: 3, 3: 4}, counts)
		assert.Equal(t, kanbanBucketCounts(t, s, u, 1, 4), counts)
	})
	t.Run("soft-deleted task in a bucket is not counted", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// Task 51 has no bucket in the fixtures; give it one so a count taken
		// straight off task_buckets would see it.
		_, err := s.Insert(&TaskBucket{TaskID: 51, ProjectViewID: 4, BucketID: 1})
		require.NoError(t, err)

		counts := listBucketCounts(t, s, u, 1, 4, true)
		assert.Equal(t, map[int64]int64{1: 11, 2: 3, 3: 4}, counts)
		assert.Equal(t, kanbanBucketCounts(t, s, u, 1, 4), counts)
	})
	t.Run("view filter applies to the counts", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		_, err := s.ID(4).Cols("filter").Update(&ProjectView{Filter: &TaskCollection{Filter: "done = true"}})
		require.NoError(t, err)

		counts := listBucketCounts(t, s, u, 1, 4, true)
		assert.Equal(t, map[int64]int64{1: 0, 2: 0, 3: 1}, counts)
		assert.Equal(t, kanbanBucketCounts(t, s, u, 1, 4), counts)
	})
	t.Run("saved filter view is counted without writing positions", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		projectID, view := createCountSavedFilter(t, s, u)

		buckets := []*Bucket{}
		require.NoError(t, s.Where("project_view_id = ?", view.ID).OrderBy("position").Find(&buckets))
		require.Len(t, buckets, 3)
		_, err := s.Where("task_id = ? AND project_view_id = ?", 1, view.ID).
			Cols("bucket_id").
			Update(&TaskBucket{BucketID: buckets[1].ID})
		require.NoError(t, err)

		// Without positions, reading the board writes them again.
		_, err = s.Where("project_view_id = ?", view.ID).Delete(&TaskPosition{})
		require.NoError(t, err)

		counts := listBucketCounts(t, s, u, projectID, view.ID, true)
		assert.Equal(t, map[int64]int64{buckets[0].ID: 2, buckets[1].ID: 1, buckets[2].ID: 0}, counts)

		positions, err := s.Where("project_view_id = ?", view.ID).Count(&TaskPosition{})
		require.NoError(t, err)
		assert.Zero(t, positions, "counting must not write task positions")

		// Last, because the board heals the positions the count left alone.
		assert.Equal(t, kanbanBucketCounts(t, s, u, projectID, view.ID), counts)
		positions, err = s.Where("project_view_id = ?", view.ID).Count(&TaskPosition{})
		require.NoError(t, err)
		assert.Equal(t, int64(3), positions)
	})
	t.Run("read share gets the counts", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		counts := listBucketCounts(t, s, u, 9, 36, true)
		assert.Equal(t, map[int64]int64{9: 1, 25: 0}, counts)
		assert.Equal(t, kanbanBucketCounts(t, s, u, 9, 36), counts)
	})
	t.Run("link share gets the counts", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		share := &LinkSharing{ID: 1, ProjectID: 1, Permission: PermissionRead}
		counts := listBucketCounts(t, s, share, 1, 4, true)
		assert.Equal(t, map[int64]int64{1: 11, 2: 3, 3: 4}, counts)
		assert.Equal(t, kanbanBucketCounts(t, s, share, 1, 4), counts)
	})
	t.Run("no access gives the same error as the plain list", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		plain := &Bucket{ProjectID: 2, ProjectViewID: 8}
		_, _, _, errPlain := plain.ReadAll(s, u, "", 0, 0)
		require.Error(t, errPlain)
		assert.True(t, IsErrGenericForbidden(errPlain))

		counted := &Bucket{ProjectID: 2, ProjectViewID: 8}
		counted.SetWithCount()
		_, _, _, errCounted := counted.ReadAll(s, u, "", 0, 0)
		require.Error(t, errCounted)
		assert.Equal(t, errPlain, errCounted)
	})
	t.Run("view of another project gives the same error as the plain list", func(t *testing.T) {
		db.LoadAndAssertFixtures(t)
		s := db.NewSession()
		defer s.Close()

		// View 8 belongs to project 2, not to project 1.
		plain := &Bucket{ProjectID: 1, ProjectViewID: 8}
		_, _, _, errPlain := plain.ReadAll(s, u, "", 0, 0)
		require.Error(t, errPlain)
		assert.True(t, IsErrProjectViewDoesNotExist(errPlain))

		counted := &Bucket{ProjectID: 1, ProjectViewID: 8}
		counted.SetWithCount()
		_, _, _, errCounted := counted.ReadAll(s, u, "", 0, 0)
		require.Error(t, errCounted)
		assert.Equal(t, errPlain, errCounted)
	})
}

// An instance admin may list the buckets of any view, but the board of a saved
// filter stays with its owner. with_count must not turn a list the admin can
// read into an error.
func TestBucket_ReadAllWithCountInstanceAdmin(t *testing.T) {
	db.LoadAndAssertFixtures(t)
	license.SetForTests([]license.Feature{license.FeatureAdminPanel})
	defer license.ResetForTests()
	s := db.NewSession()
	defer s.Close()

	_, err := s.ID(int64(2)).Cols("is_admin").Update(&user.User{IsAdmin: true})
	require.NoError(t, err)
	admin := &user.User{ID: 2, IsAdmin: true}

	t.Run("project of another user gets the real counts", func(t *testing.T) {
		counts := listBucketCounts(t, s, admin, 1, 4, true)
		assert.Equal(t, map[int64]int64{1: 11, 2: 3, 3: 4}, counts)
		assert.Equal(t, kanbanBucketCounts(t, s, admin, 1, 4), counts)
	})
	t.Run("saved filter of another user answers like the plain list", func(t *testing.T) {
		projectID, view := createCountSavedFilter(t, s, &user.User{ID: 1})

		board := &TaskCollection{ProjectID: projectID, ProjectViewID: view.ID}
		_, _, _, err := board.ReadAll(s, admin, "", 0, 0)
		require.Error(t, err)
		assert.True(t, IsErrGenericForbidden(err), "the board of the filter is the owner's only")

		plain := listBucketCounts(t, s, admin, projectID, view.ID, false)
		require.Len(t, plain, 3)
		assert.Equal(t, plain, listBucketCounts(t, s, admin, projectID, view.ID, true))
	})
}

// apiPermissionForRoute returns the permission CollectRoutesForAPITokenUsage
// filed the route under, whatever name it ended up with.
func apiPermissionForRoute(t *testing.T, method, path string) (group, permission string) {
	t.Helper()
	for _, table := range []map[string]APITokenRoute{apiTokenRoutes, apiTokenRoutesV2} {
		for g, perms := range table {
			for p, rd := range perms {
				if rd.Method == method && rd.Path == path {
					return g, p
				}
			}
		}
	}
	require.FailNow(t, "route not collected", "%s %s", method, path)
	return "", ""
}

// with_count tells how many tasks each bucket holds, so an API token needs a
// scope which already reads the view's tasks; listing buckets is not enough.
func TestCanDoAPIRoute_BucketCountScope(t *testing.T) {
	v1Routes, v2Routes := apiTokenRoutes, apiTokenRoutesV2
	defer func() {
		apiTokenRoutes, apiTokenRoutesV2 = v1Routes, v2Routes
	}()
	apiTokenRoutes = make(map[string]APITokenRoute)
	apiTokenRoutesV2 = make(map[string]APITokenRoute)

	type route struct{ method, path string }
	list := route{http.MethodGet, bucketListRouteV2}
	board := route{http.MethodGet, "/api/v2/projects/:project/views/:view/buckets/tasks"}
	viewTasks := route{http.MethodGet, "/api/v2/projects/:project/views/:view/tasks"}
	// Collected first on purpose: moving a task takes the name views_buckets_tasks
	// and the board gets another one, so the check must follow the route.
	move := route{http.MethodPost, "/api/v2/projects/:project/views/:view/buckets/:bucket/tasks"}

	for _, r := range []route{move, list, board, viewTasks} {
		CollectRoutesForAPITokenUsage(echo.RouteInfo{Method: r.method, Path: r.path}, true)
	}
	moveGroup, movePermission := apiPermissionForRoute(t, move.method, move.path)
	require.Equal(t, "projects", moveGroup)
	require.Equal(t, "views_buckets_tasks", movePermission)

	tokenFor := func(routes ...route) *APIToken {
		perms := APIPermissions{}
		for _, r := range routes {
			group, permission := apiPermissionForRoute(t, r.method, r.path)
			perms[group] = append(perms[group], permission)
		}
		return &APIToken{APIPermissions: perms}
	}

	e := echo.New()
	can := func(url string, token *APIToken) bool {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		c := e.NewContext(req, httptest.NewRecorder())
		return CanDoAPIRoute(c, token)
	}

	listOnly := tokenFor(list)

	t.Run("a bucket list token keeps the plain list", func(t *testing.T) {
		for _, query := range []string{"", "?with_count=false", "?with_count=0", "?with_count="} {
			assert.True(t, can(bucketListRouteV2+query, listOnly), "query %q", query)
		}
	})
	t.Run("a bucket list token cannot read the counts", func(t *testing.T) {
		for _, query := range []string{
			"?with_count=true",
			"?with_count=1",
			"?with_count=TRUE",
			"?with_count=false&with_count=true",
			"?with_count=maybe",
		} {
			assert.False(t, can(bucketListRouteV2+query, listOnly), "query %q", query)
		}
	})
	t.Run("moving tasks is not reading them", func(t *testing.T) {
		assert.False(t, can(bucketListRouteV2+"?with_count=true", tokenFor(list, move)))
	})
	t.Run("a scope which reads the view's tasks unlocks the counts", func(t *testing.T) {
		assert.True(t, can(bucketListRouteV2+"?with_count=true", tokenFor(list, board)))
		assert.True(t, can(bucketListRouteV2+"?with_count=true", tokenFor(list, viewTasks)))
	})
	t.Run("a task scope alone does not open the bucket list", func(t *testing.T) {
		assert.False(t, can(bucketListRouteV2+"?with_count=true", tokenFor(board)))
	})
}
