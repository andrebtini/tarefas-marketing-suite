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
	"xorm.io/builder"
	"xorm.io/xorm"
)

// GetUserIDsSharingProjectOrTeam returns the users who share at least one project
// or one team with userID, userID itself left out. Two users share a project when
// both can reach it, whether through ownership, a direct share or a team share,
// and whether granted on the project itself or on one of its parents.
func GetUserIDsSharingProjectOrTeam(s *xorm.Session, userID int64) (map[int64]bool, error) {
	access, err := getProjectAccessForUser(s, userID)
	if err != nil {
		return nil, err
	}
	// A grant on a parent reaches every child, so whoever holds one on an ancestor
	// of a reachable project shares that project too.
	ancestorIDs, err := getUnreachableAncestorProjectIDs(s, access)
	if err != nil {
		return nil, err
	}
	reachable := func(column string) builder.Cond {
		return builder.Or(access.cond(column), builder.In(column, ancestorIDs))
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
					reachable("team_projects.project_id"),
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

// getUnreachableAncestorProjectIDs walks up from the reachable projects, one level
// per query, and returns the ancestors that are not reachable themselves.
func getUnreachableAncestorProjectIDs(s *xorm.Session, access *projectAccess) ([]int64, error) {
	known := make(map[int64]bool, len(access.sortedIDs))
	for _, id := range access.sortedIDs {
		known[id] = true
	}

	ancestors := []int64{}
	level := access.cond("id")
	for {
		parents := []int64{}
		err := s.Table("projects").
			Select("parent_project_id").
			Where(builder.And(level, builder.Gt{"parent_project_id": 0})).
			Find(&parents)
		if err != nil {
			return nil, err
		}

		next := []int64{}
		for _, id := range parents {
			if !known[id] {
				known[id] = true
				next = append(next, id)
			}
		}
		if len(next) == 0 {
			return ancestors, nil
		}
		ancestors = append(ancestors, next...)
		level = builder.In("id", next)
	}
}
