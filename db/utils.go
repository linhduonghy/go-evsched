package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TimeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func StringToPgText(t string) pgtype.Text {
	return pgtype.Text{String: t, Valid: true}
}

func PgTextToString(t pgtype.Text) string {
	return t.String
}

func PgTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	return t.Time
}

func PgInt8ToInt64(t pgtype.Int8) *int64 {
	if !t.Valid {
		return nil
	}
	return &t.Int64
}

func Int64ToPgInt8(t int64) pgtype.Int8 {
	return pgtype.Int8{Int64: t, Valid: true}
}
