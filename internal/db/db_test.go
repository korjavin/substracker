package db

import (
	"database/sql"
	"testing"
)

func TestMigrationAddsUserID(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	err = Migrate(db)
	if err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	tables := []string{"subscriptions", "webpush_subscriptions", "telegram_chats"}
	for _, table := range tables {
		rows, err := db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			t.Fatalf("failed to query table info for %s: %v", table, err)
		}

		found := false
		for rows.Next() {
			var cid int
			var name string
			var typeStr string
			var notnull int
			var dflt_value sql.NullString
			var pk int

			if err := rows.Scan(&cid, &name, &typeStr, &notnull, &dflt_value, &pk); err != nil {
				t.Fatalf("failed to scan table info: %v", err)
			}
			if name == "user_id" {
				found = true
				if typeStr != "INTEGER" {
					t.Errorf("expected type INTEGER for user_id in %s, got %s", table, typeStr)
				}
				if notnull != 1 {
					t.Errorf("expected notnull=1 for user_id in %s", table)
				}
				if dflt_value.String != "0" {
					t.Errorf("expected default=0 for user_id in %s, got %s", table, dflt_value.String)
				}
			}
		}
		rows.Close()

		if !found {
			t.Errorf("user_id column not found in %s", table)
		}
	}
}
