package lock1

import (
	"fmt"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stvp/tempredis"
)

var (
	pool *redis.Pool
)

func TestMain(m *testing.M) {
	// rds, err := miniredis.Run()
	// if err != nil {
	// 	fmt.Errorf("miniredis start fail %v", err)
	// }
	server, err := tempredis.Start(tempredis.Config{})
	if err != nil {
		fmt.Errorf("miniredis start fail %v", err)
	}
	pool = newPool(server.Socket())
	m.Run()
}

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) { return redis.Dial("unix", addr) },
	}
}

func logFuc(t *testing.T) func(...interface{}) {
	return func(args ...interface{}) {
		tstr := time.Now().Format("2006-01-02 15:04:05")
		t.Logf("[%s] %v \n", tstr, args)
	}
}

func TestTryLock(t *testing.T) {
	l := NewLock("TestTryLock", pool, 3)
	log := logFuc(t)

	res, err := l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("first tryLock expect true but get false")
	}
	log("case 1 first lock pass")
	res, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == true {
		t.Fatalf("second tryLock expect false but get true")
	}
	log("case 2 second lock pass")

	log("waiting for expiration....")
	// 等待过期
	time.Sleep(time.Millisecond * 3500)

	res, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("tryLock after expiration expect true but get false")
	}
	log("case 3 lock expire pass")
}

func TestRelease(t *testing.T) {
	l := NewLock("TestRelease", pool, 3)
	log := logFuc(t)
	res, err := l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("first tryLock expect true but get false")
	}
	log("case 1 first lock pass")
	_, err = l.Release()
	if err != nil {
		t.Fatal(err)
	}
	log("lock released")
	res, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("tryLock after realse expected true but get false")
	}
	log("case 1 lock after release pass")
}
