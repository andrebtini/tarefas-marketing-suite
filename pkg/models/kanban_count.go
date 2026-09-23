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
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// SetWithCount makes ReadAll fill Count on every bucket. Only the v2 bucket
// list sets it (with_count=true); v1 keeps listing buckets without counting.
func (b *Bucket) SetWithCount() {
	b.withCount = true
}

// addTaskCountsToBuckets sets each bucket's Count to the number the kanban
// board reports for it. It runs the board's own read path in count-only mode,
// so the view filter, saved filters, link shares and soft-deleted tasks are
// handled exactly like on the board, at one COUNT per bucket and without
// loading any task. Only manual buckets live in the buckets table, so views in
// another bucket mode keep Count at 0.
func addTaskCountsToBuckets(s *xorm.Session, a web.Auth, view *ProjectView, buckets []*Bucket) error {
	if len(buckets) == 0 || view.BucketConfigurationMode != BucketConfigurationModeManual {
		return nil
	}

	tc := &TaskCollection{
		ProjectID:     view.ProjectID,
		ProjectViewID: view.ID,
		countOnly:     true,
	}
	result, _, _, err := tc.ReadAll(s, a, "", 0, 0)
	if err != nil {
		return err
	}

	// A view filter on bucket_id makes the board a flat task list, which has
	// no per-bucket number to report.
	board, is := result.([]*Bucket)
	if !is {
		return nil
	}

	counts := make(map[int64]int64, len(board))
	for _, bb := range board {
		counts[bb.ID] = bb.Count
	}
	for _, bb := range buckets {
		bb.Count = counts[bb.ID]
	}

	return nil
}
