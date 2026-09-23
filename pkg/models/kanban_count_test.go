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
	"code.vikunja.io/api/pkg/web"

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

// Fixtures: project 1 (user 1) has kanban view 4 with buckets 1, 2 and 3. Bucket
// 3 is the view's done bucket and holds task 2, the only done task. Task 51 of
// project 1 is soft-deleted. Project 9 is shared read-only with user 1, and its
// kanban view 36 has bucket 9 (task 18) and bucket 25 (empty). Project 2 belongs
// to user 3 and is not shared with user 1.
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
