package errors

import "github.com/lib/pq"

// IsUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (error code 23505).
func IsUniqueViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505"
	}
	return false
}

// IsForeignKeyViolation reports whether err is a PostgreSQL foreign-key
// constraint violation (error code 23503).
func IsForeignKeyViolation(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23503"
	}
	return false
}
