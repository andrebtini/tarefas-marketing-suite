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
	"code.vikunja.io/api/pkg/user"

	"xorm.io/builder"
	"xorm.io/xorm"
)

// GetUserIDsSharingProjectOrTeam returns the users who share at least one project
// or one team with userID, userID itself left out. Two users share a project when
// both can reach it, whether through ownership, a direct share or a team share,
// and whether granted on the project itself or on one of its parents.
// The relation is symmetric and holds every user GetUserIDsVisibleTo can return
// for userID, so whoever is not in it can never see userID either.
func GetUserIDsSharingProjectOrTeam(s *xorm.Session, userID int64) (map[int64]bool, error) {
	return getUserIDsInCommon(s, userID, false)
}

// GetUserIDsVisibleTo returns the users viewerID works with: the members of their
// teams and, on every project they reach, the users ListUsersFromProject shows
// them there. Outside the projects they administer that leaves out the members
// of teams shared on the project that they do not belong to, so unlike
// GetUserIDsSharingProjectOrTeam the relation is not symmetric.
func GetUserIDsVisibleTo(s *xorm.Session, viewerID int64) (map[int64]bool, error) {
	return getUserIDsInCommon(s, viewerID, true)
}

func getUserIDsInCommon(s *xorm.Session, userID int64, asProjectUserList bool) (map[int64]bool, error) {
	access, err := getProjectAccessForUser(s, userID)
	if err != nil {
		return nil, err
	}
	// A grant on a parent reaches every child, so whoever holds one on an ancestor
	// of a reachable project shares that project too.
	parents, ancestorIDs, err := getProjectAncestry(s, access)
	if err != nil {
		return nil, err
	}
	reachable := func(column string) builder.Cond {
		return builder.Or(access.cond(column), builder.In(column, ancestorIDs))
	}
	teamShared := reachable
	// An instance admin administers every project, where every team share is listed.
	if asProjectUserList && !isInstanceAdmin(s, &user.User{ID: userID}) {
		listingIDs := adminProjectChainIDs(access, parents)
		teamShared = func(column string) builder.Cond {
			return builder.In(column, listingIDs)
		}
	}

	// Each query runs before the next is built: they all share the session's statement.
	queries := []func(found *[]int64) error{
		func(found *[]int64) error {
			return s.Table("projects").
				Select("owner_id").
				Where(reachable("id")).
				Find(found)
		},
		func(found *[]int64) error {
			return s.Table("users_projects").
				Select("user_id").
				Where(builder.And(
					reachable("project_id"),
					builder.In("permission", PermissionRead, PermissionWrite, PermissionAdmin),
				)).
				Find(found)
		},
		func(found *[]int64) error {
			return s.Table("team_members").
				Select("team_members.user_id").
				Join("INNER", "team_projects", "team_projects.team_id = team_members.team_id").
				Where(builder.And(
					teamShared("team_projects.project_id"),
					builder.In("team_projects.permission", PermissionRead, PermissionWrite, PermissionAdmin),
				)).
				Find(found)
		},
		func(found *[]int64) error {
			return s.Table("team_members").
				Select("user_id").
				Where(builder.In("team_id", builder.Select("team_id").From("team_members").Where(builder.Eq{"user_id": userID}))).
				Find(found)
		},
	}

	ids := make(map[int64]bool)
	for _, query := range queries {
		found := []int64{}
		if err := query(&found); err != nil {
			return nil, err
		}
		for _, id := range found {
			if id > 0 {
				ids[id] = true
			}
		}
	}

	delete(ids, userID)
	return ids, nil
}

// ListUsersFromProject shows a non-admin only the members of the teams they are
// in, who are their teammates anyway. So team shares only add someone on the
// projects the user administers and on the parents those projects inherit from,
// the same chain ListUsersFromProject walks up.
func adminProjectChainIDs(access *projectAccess, parents map[int64]int64) []int64 {
	listed := make(map[int64]bool)
	ids := []int64{}
	for _, id := range access.sortedIDs {
		if access.permissions[id] != PermissionAdmin {
			continue
		}
		for current := id; current > 0 && !listed[current]; current = parents[current] {
			listed[current] = true
			ids = append(ids, current)
		}
	}
	return ids
}

type projectParentRow struct {
	ID              int64 `xorm:"id"`
	ParentProjectID int64 `xorm:"parent_project_id"`
}

// getProjectAncestry walks up from the reachable projects, one level per query. It
// returns the parent of every project it passed and the ancestors that are not
// reachable themselves.
func getProjectAncestry(s *xorm.Session, access *projectAccess) (parents map[int64]int64, ancestorIDs []int64, err error) {
	known := make(map[int64]bool, len(access.sortedIDs))
	for _, id := range access.sortedIDs {
		known[id] = true
	}

	parents = make(map[int64]int64)
	ancestorIDs = []int64{}
	level := access.cond("id")
	for {
		rows := []*projectParentRow{}
		err = s.Table("projects").
			Select("id, parent_project_id").
			Where(builder.And(level, builder.Gt{"parent_project_id": 0})).
			Find(&rows)
		if err != nil {
			return nil, nil, err
		}

		next := []int64{}
		for _, row := range rows {
			parents[row.ID] = row.ParentProjectID
			if !known[row.ParentProjectID] {
				known[row.ParentProjectID] = true
				next = append(next, row.ParentProjectID)
			}
		}
		if len(next) == 0 {
			return parents, ancestorIDs, nil
		}
		ancestorIDs = append(ancestorIDs, next...)
		level = builder.In("id", next)
	}
}
