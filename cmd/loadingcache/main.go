package main

import (
	"database/sql"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goldstd/loadingcache"
	"github.com/pkg/errors"
	"github.com/xo/dburl"
)

var (
	expire     = flag.Duration("expire", 5*time.Second, "expiration")
	query      = flag.String("query", "select v from gcache where k = ?", "query SQL")
	dsn        = flag.String("dsn", "mysql://root:root@127.0.0.1:3306/gcache", "DSN")
	port       = flag.Int("port", 8080, "Port to listen on")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
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

	defer profiling()()

	dbLoader := &DBLoader{DB: db, Query: *query}

	cache = loadingcache.Config{
		AsyncLoad:        true,
		Load:             dbLoader,
		ExpireAfterWrite: *expire,
	}.Build()

	http.HandleFunc("/", httpHandle)
	server := &http.Server{
		Addr: ":" + strconv.Itoa(*port),
	}

	go signalCapture(server)

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
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
	} else if str, ok := value.(string); ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(strconv.Quote(str)))
	} else {
		log.Printf("invalid val: %+v", value)
	}
}

type DBLoader struct {
	DB    *sql.DB
	Query string
}

func (d *DBLoader) Load(key any, cache loadingcache.Cache) (any, error) {
	row, err := d.DB.Query(d.Query, key)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var val sql.NullString

	if row.Next() {
		if err := row.Scan(&val); err != nil {
			return nil, err
		}
	}

	return val.String, nil
}

func signalCapture(closeables ...io.Closer) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	for _, cl := range closeables {
		if cl != nil {
			if err := cl.Close(); err != nil {
				log.Printf("close %t: %v", cl, err)
			}
		}
	}
}

func profiling() func() {
	var deferFuncs []func()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		deferFuncs = append(deferFuncs, func() {
			_ = f.Close()
		})
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		deferFuncs = append(deferFuncs, pprof.StopCPUProfile)
	}

	if *memprofile != "" {
		deferFuncs = append(deferFuncs, func() {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			defer f.Close()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
		})
	}

	return func() {
		for i := len(deferFuncs) - 1; i >= 0; i-- {
			deferFuncs[i]()
		}
	}
}
