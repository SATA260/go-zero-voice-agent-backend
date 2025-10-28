package tool

import "database/sql"

func StringToNullString(s string) sql.NullString {
    return sql.NullString{
        String: s,
        Valid:  s != "",
    }
}

func NullStringToString(ns sql.NullString) string {
    if ns.Valid {
        return ns.String
    }
    return ""
}

func Int64ToNullInt64(i int64) sql.NullInt64 {
    return sql.NullInt64{
        Int64: i,
        Valid: true,
    }
}

func NullInt64ToInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}

func Float64ToNullFloat64(f float64) sql.NullFloat64 {
    return sql.NullFloat64{
        Float64: f,
        Valid:   true,
    }
}
func NullFloat64ToFloat64(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}