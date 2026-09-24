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

package migration

import (
	"strings"
	"testing"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// usersBefore20260923202143 is users as it stood before the migration, with the
// unique username and the plain indexes a partial sync must not drop (#3244).
type usersBefore20260923202143 struct {
	ID                  int64  `xorm:"bigint autoincr not null unique pk"`
	Name                string `xorm:"text null"`
	Username            string `xorm:"varchar(250) not null unique"`
	Email               string `xorm:"varchar(250) null"`
	DiscoverableByName  bool   `xorm:"bool default false index"`
	DiscoverableByEmail bool   `xorm:"bool default false index"`
	DefaultProjectID    int64  `xorm:"bigint null index"`
	BotOwnerID          int64  `xorm:"bigint null index"`
}

func (usersBefore20260923202143) TableName() string {
	return "users"
}

type usersAfter20260923202143 struct {
	ID       int64  `xorm:"bigint autoincr not null unique pk"`
	Username string `xorm:"varchar(250) not null unique"`
	JobTitle string `xorm:"varchar(100) null"`
}

func (usersAfter20260923202143) TableName() string {
	return "users"
}

func TestAddUserJobTitle20260923202143(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	table := usersBefore20260923202143{}
	t.Cleanup(func() {
		// x is the process-global test engine: leave the full users table behind.
		require.NoError(t, x.DropTables(table))
		require.NoError(t, x.Sync2(user.GetTables()...))
	})
	require.NoError(t, x.DropTables(table))
	require.NoError(t, x.Sync2(table))

	existing := &usersBefore20260923202143{ID: 1, Name: "Existing", Username: "existing20260923202143"}
	_, err = x.Insert(existing)
	require.NoError(t, err)

	before := usersTable20260923202143(t, x)
	require.Nil(t, before.GetColumn("job_title"))
	require.NotEmpty(t, before.Indexes)
	require.NoError(t, addUserJobTitle20260923202143(x))

	after := usersTable20260923202143(t, x)
	require.NotNil(t, after.GetColumn("job_title"))
	for _, column := range before.ColumnsSeq() {
		require.NotNilf(t, after.GetColumn(column), "migration dropped column %s", column)
	}
	for name, index := range before.Indexes {
		preserved, found := after.Indexes[name]
		require.Truef(t, found, "migration dropped index %s", name)
		require.Equal(t, index.Type, preserved.Type)
		require.Equal(t, index.Cols, preserved.Cols)
	}

	got := &usersAfter20260923202143{}
	found, err := x.ID(existing.ID).Get(got)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, existing.Username, got.Username)
	require.Empty(t, got.JobTitle, "users that existed before the migration must have no job title")

	// The model allows 100 characters, not bytes, so the column must hold them on every database.
	title := strings.Repeat("é", 100)
	_, err = x.ID(existing.ID).Cols("job_title").Update(&usersAfter20260923202143{JobTitle: title})
	require.NoError(t, err)
	titled := &usersAfter20260923202143{}
	_, err = x.ID(existing.ID).Get(titled)
	require.NoError(t, err)
	require.Equal(t, title, titled.JobTitle)
}

func usersTable20260923202143(t *testing.T, x *xorm.Engine) *schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, table := range tables {
		if table.Name == "users" {
			return table
		}
	}
	t.Fatal("users table not found")
	return nil
}
