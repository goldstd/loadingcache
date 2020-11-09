package loadingcache_test

import (
	"fmt"
	"hash/fnv"

	"github.com/Hartimer/loadingcache"
	"github.com/pkg/errors"
)

type testRemovalListener struct {
	lastRemovalNotification loadingcache.RemovalNotification
}

func (t *testRemovalListener) Listener(notification loadingcache.RemovalNotification) {
	t.lastRemovalNotification = notification
}

// testLoadFunc provides a configurable loading function that may fail
type testLoadFunc struct {
	fail bool
}

func (t *testLoadFunc) LoadFunc(key interface{}) (interface{}, error) {
	if t.fail {
		return nil, errors.New("failing on request")
	}
	return fmt.Sprint(key), nil
}

// stringHashCodeFunc is a test hash code function for strings which uses fnv.New32a
var stringHashCodeFunc = func(k interface{}) int {
	h := fnv.New32a()
	h.Write([]byte(k.(string)))
	return int(h.Sum32())
}

// intHashCodeFunc is a test hash code function for ints which just passes them through
var intHashCodeFunc = func(k interface{}) int {
	return k.(int)
}
