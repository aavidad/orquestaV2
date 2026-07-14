package sqlite

import (
	"database/sql"
	"time"
)

func storedTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.Round(0).UTC().UnixNano()
}

func requiredTime(value time.Time) int64 {
	return value.Round(0).UTC().UnixNano()
}

func restoredTime(value sql.NullInt64) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return time.Unix(0, value.Int64).UTC()
}
