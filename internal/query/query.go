package query

import (
	"github.com/jinzhu/gorm"
	"github.com/njhsi/8ackyard/internal/entity"
	"github.com/njhsi/8ackyard/internal/event"
)

var log = event.Log

const (
	MySQL   = "mysql"
	SQLite3 = "sqlite3"
)

// Cols represents a list of database columns.
type Cols []string

// Query searches given an originals path and a db instance.
type Query struct {
	db *gorm.DB
}

// SearchCount is the total number of search hits.
type SearchCount struct {
	Total int
}

// New returns a new Query type with a given path and db instance.
func New(db *gorm.DB) *Query {
	q := &Query{
		db: db,
	}

	return q
}

// Db returns a database connection instance.
func Db() *gorm.DB {
	return entity.Db()
}

// UnscopedDb returns an unscoped database connection instance.
func UnscopedDb() *gorm.DB {
	return entity.Db().Unscoped()
}

// DbDialect returns the sql dialect name.
func DbDialect() string {
	return Db().Dialect().GetName()
}
