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
	"testing"

	"code.vikunja.io/api/pkg/db"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

type bucketsBefore20260923201930 struct {
	ID            int64  `xorm:"bigint autoincr not null unique pk"`
	Title         string `xorm:"text not null"`
	ProjectViewID int64  `xorm:"bigint not null"`
	CreatedByID   int64  `xorm:"bigint not null"`
}

func (bucketsBefore20260923201930) TableName() string {
	return "buckets"
}

type bucketsAfter20260923201930 struct {
	ID       int64  `xorm:"bigint autoincr not null unique pk"`
	Title    string `xorm:"text not null"`
	HexColor string `xorm:"varchar(6) null"`
}

func (bucketsAfter20260923201930) TableName() string {
	return "buckets"
}

func TestAddBucketHexColor20260923201930(t *testing.T) {
	x, err := db.CreateTestEngine()
	require.NoError(t, err)

	table := bucketsBefore20260923201930{}
	t.Cleanup(func() {
		require.NoError(t, x.DropTables(table))
	})
	require.NoError(t, x.DropTables(table))
	require.NoError(t, x.Sync2(table))

	existing := &bucketsBefore20260923201930{ID: 1, Title: "Existing", ProjectViewID: 4, CreatedByID: 1}
	_, err = x.Insert(existing)
	require.NoError(t, err)

	before := bucketsTable20260923201930(t, x)
	require.NotEmpty(t, before.Indexes)
	require.NoError(t, addBucketHexColor20260923201930(x))

	after := bucketsTable20260923201930(t, x)
	require.NotNil(t, after.GetColumn("hex_color"))
	for _, column := range before.ColumnsSeq() {
		require.NotNilf(t, after.GetColumn(column), "migration dropped column %s", column)
	}
	for name, index := range before.Indexes {
		preserved, found := after.Indexes[name]
		require.Truef(t, found, "migration dropped index %s", name)
		require.Equal(t, index.Type, preserved.Type)
		require.Equal(t, index.Cols, preserved.Cols)
	}

	got := &bucketsAfter20260923201930{}
	found, err := x.ID(existing.ID).Get(got)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, existing.Title, got.Title)
	require.Empty(t, got.HexColor, "buckets that existed before the migration must have no color")

	_, err = x.ID(existing.ID).Cols("hex_color").Update(&bucketsAfter20260923201930{HexColor: "e8e0ff"})
	require.NoError(t, err)
	colored := &bucketsAfter20260923201930{}
	_, err = x.ID(existing.ID).Get(colored)
	require.NoError(t, err)
	require.Equal(t, "e8e0ff", colored.HexColor)
}

func bucketsTable20260923201930(t *testing.T, x *xorm.Engine) *schemas.Table {
	t.Helper()
	tables, err := x.DBMetas()
	require.NoError(t, err)
	for _, table := range tables {
		if table.Name == "buckets" {
			return table
		}
	}
	t.Fatal("buckets table not found")
	return nil
}
