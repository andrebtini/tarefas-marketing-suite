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
	"strconv"

	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/web"

	"github.com/labstack/echo/v5"
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
	if IsErrGenericForbidden(err) {
		// The list already passed its own read check, yet the board can still
		// refuse: an instance admin may list the buckets of someone else's
		// saved filter but not read its tasks. The list must answer as it
		// does without with_count, so those buckets keep Count at 0.
		return nil
	}
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

// bucketListRouteV2 is the bucket list route, the only one taking with_count.
const bucketListRouteV2 = "/api/v2/projects/:project/views/:view/buckets"

// bucketCountTaskRoutes already return the tasks of a view, so a token allowed
// on one of them may also read how many tasks each bucket holds.
var bucketCountTaskRoutes = []string{
	"/api/v2/projects/:project/views/:view/buckets/tasks",
	"/api/v2/projects/:project/views/:view/tasks",
}

// bucketCountScopeSatisfied keeps with_count behind a task scope for API
// tokens. The counts describe tasks, so listing buckets alone must not reveal
// them, the same way expand=comment_count needs the comments scope. The check
// follows the routes rather than a permission name, because the name
// views_buckets_tasks is shared with the route that moves a task.
func bucketCountScopeSatisfied(c *echo.Context, token *APIToken, path, method string) bool {
	if method != http.MethodGet || path != bucketListRouteV2 || !bucketCountRequested(c) {
		return true
	}

	for _, route := range bucketCountTaskRoutes {
		if tokenAuthorizesRoute(token, route, http.MethodGet) {
			return true
		}
	}

	log.Debugf("[auth] Token %d tried to read bucket counts on %s which is not covered by its permissions %v",
		token.ID, path, token.APIPermissions)
	return false
}

// bucketCountRequested reports whether the request may make the route count.
// Every value is checked, not only the one the route reads, and a value the
// route would reject counts as a request too, so no spelling slips through.
func bucketCountRequested(c *echo.Context) bool {
	for _, raw := range c.Request().URL.Query()["with_count"] {
		if raw == "" {
			continue
		}
		wanted, err := strconv.ParseBool(raw)
		if err != nil || wanted {
			return true
		}
	}
	return false
}
