package main

import (
	"database/sql"
	"flag"
	"github.com/goldstd/loadingcache"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/xo/dburl"
)

var (
	expire = flag.Duration("expire", 15*time.Second, "expiration")
	query  = flag.String("query", "select v from gcache where k = ?", "query SQL")
	dsn    = flag.String("dsn", "mysql://root:root@tcp(127.0.0.1:3306)/gcache", "DSN")
	port   = flag.Int("port", 8080, "Port to listen on")
)

var cache loadingcache.Cache

func main() {
	flag.Parse()

	db, err := dburl.Open(*dsn)
	if err != nil {
		log.Printf("Error parsing DSN: %v", err)
		return
	}

	defer db.Close()

	dbLoader := &DBLoader{DB: db, Query: *query}

	cache = loadingcache.Config{
		AsyncLoad:        true,
		Load:             dbLoader,
		ExpireAfterWrite: *expire,
	}.Build()

	http.HandleFunc("/", httpHandle)
	if err := http.ListenAndServe(":"+strconv.Itoa(*port), nil); err != nil {
		log.Printf("Error listening on port %d: %v", *port, err)
	}

	log.Printf("exiting")
}

func httpHandle(w http.ResponseWriter, r *http.Request) {
	key := "k1"
	if strings.HasPrefix(r.URL.Path, "/key/") {
		key = r.URL.Path[5:]
	}

	value, err := cache.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(value.(string)))
	}
}

type DBLoader struct {
	DB    *sql.DB
	Query string
}

func (d *DBLoader) Load(key any) (any, error) {
	row := d.DB.QueryRow(d.Query, key)
	var value string
	err := row.Scan(&value)
	return value, err
}
