package dbloader

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/goldstd/loadingcache"
)

type DBLoader struct {
	StructType reflect.Type
	DB         *sql.DB
	Query      string
}

func NewDBLoader(db *sql.DB, query string, structType any) *DBLoader {
	return &DBLoader{
		DB:         db,
		Query:      query,
		StructType: reflect.TypeOf(structType),
	}
}

func (d *DBLoader) Load(key any, cache loadingcache.Cache) (any, error) {
	val := reflect.New(d.StructType)
	ctx := context.Background()
	if err := sqlscan.Get(ctx, d.DB, val.Interface(), d.Query, key); err != nil {
		return nil, err
	}

	return val.Elem().Interface(), nil
}
