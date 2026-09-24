package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// Sentinel errors shared by all repositories.
var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate record")
)

func isDuplicate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || // PostgreSQL
		strings.Contains(msg, "duplicate entry") || // MySQL
		strings.Contains(msg, "unique constraint failed") // SQLite
}

func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if isDuplicate(err) {
		return ErrDuplicate
	}
	return err
}
