package lock

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stvp/tempredis"
)

var (
	pool *redis.Pool
)

func getTime() string {
	now := time.Now()
	tstr := now.Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s.%d] ", tstr, now.Nanosecond())
}

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

func TestTryLock(t *testing.T) {
	l := NewLock("TestTryLock", pool, &LockOptions{TimeoutSec: 3})

	res, err := l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("first tryLock expect true but get false")
	}
	t.Log(getTime(), "case 1 first lock pass")
	res, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == true {
		t.Fatalf("second tryLock expect false but get true")
	}
	t.Log(getTime(), "case 2 second lock pass")

	t.Log(getTime(), "waiting for expiration....")
	// 等待过期
	time.Sleep(time.Millisecond * 3500)

	res, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("tryLock after expiration expect true but get false")
	}
	t.Log(getTime(), "case 3 lock expire pass")
}

func TestRelease(t *testing.T) {
	l := NewLock("TestRelease", pool, &LockOptions{TimeoutSec: 3})
	res, err := l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("first tryLock expect true but get false")
	}
	t.Log(getTime(), "case 1 first lock pass")
	_, err = l.Release()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(getTime(), "lock released")
	res, err = l.TryLock()
	if err != nil {
		t.Fatal(err)
	}
	if res == false {
		t.Fatalf("tryLock after realse expected true but get false")
	}
	t.Log(getTime(), "case 1 lock after release pass")
}

func TestLock(t *testing.T) {
	l := NewLock("TestLock", pool, &LockOptions{TimeoutSec: 3})
	err := l.Lock()
	if err != nil {
		t.Fatal(err)
	}
	err = l.Lock()
	if err != ErrMaxTries {
		t.Fatal("expect lock max retry")
	}
	l.Release()

	wg := sync.WaitGroup{}
	paraSize := 5
	var successNum int32 = 0
	wg.Add(paraSize)
	for i := 0; i < paraSize; i++ {
		go func() {
			err := l.Lock()
			if err == nil {
				atomic.AddInt32(&successNum, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	t.Log(getTime(), "successNum", successNum)
	if successNum > 1 {
		t.Fatal("parallel lock success num > 1")
	}
}

func TestLockTimeout(t *testing.T) {
	l := NewLock("TestLockTimeout", pool, &LockOptions{TimeoutSec: 3})
	err := l.Lock()
	if err != nil {
		t.Fatal(err)
	}

	l1 := NewLock("TestLockTimeout", pool, &LockOptions{TimeoutSec: 3, GetLockTmeout: time.Millisecond * 100})
	err = l1.Lock()
	if err != ErrGetLockTimeout {
		t.Fatal("expected ErrGetLockTimeout")
	}
}

func TestLockAllSuccess(t *testing.T) {
	l := NewLock("TestLockAllSuccess", pool, &LockOptions{TimeoutSec: 3})
	wg := sync.WaitGroup{}
	paraSize := 5
	var successNum int32 = 0
	wg.Add(paraSize)
	for i := 0; i < paraSize; i++ {
		go func(seq int) {
			err := l.Lock()
			if err == nil {
				atomic.AddInt32(&successNum, 1)
			}
			t.Log(getTime(), "lock success", seq)
			l.Release()
			wg.Done()
		}(i)
	}
	wg.Wait()
	t.Log(getTime(), "successNum", successNum)
	if successNum != int32(paraSize) {
		t.Fatal("parallel lock success num > 1")
	}
}

func TestReleaseOthersLock(t *testing.T) {
	l1 := NewLock("TestReleaseOthersLock", pool, &LockOptions{TimeoutSec: 3})
	l2 := NewLock("TestReleaseOthersLock", pool, &LockOptions{TimeoutSec: 3})

	err := l1.Lock()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(getTime(), "l1 lock success")

	t.Log(getTime(), "l2 try to release lock")
	ok, err := l2.Release()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("l2 released l1's lock")
	}
	t.Log(getTime(), "l2 can't release l1's lock")
	ok, err = l1.Release()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("l1 released failed")
	}

	t.Log(getTime(), "l1 release lock success")
}

func TestLeftSec(t *testing.T) {
	l1 := NewLock("TestLeftSec", pool, &LockOptions{TimeoutSec: 3})
	res, err := l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	if res != int64(-2) {
		t.Fatal("not exist key LeftSec val err")
	}

	err = l1.Lock()
	if err != nil {
		t.Fatal(err)
	}
	res, err = l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("left seconds: %d\n", res)
	if res < 2 || res > 3 {
		t.Fatal("LeftSec val wrong")
	}
}

func TestExtend(t *testing.T) {
	l1 := NewLock("TestExtend", pool, &LockOptions{TimeoutSec: 3})
	l2 := NewLock("TestExtend", pool, &LockOptions{TimeoutSec: 3})
	err := l1.Lock()
	if err != nil {
		t.Fatal(err)
	}

	ok, err := l1.Extend(5)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("extend fail")
	}
	res, err := l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("left seconds: %d\n", res)
	if res < 4 || res > 5 {
		t.Fatal("LeftSec val wrong")
	}

	ok, err = l2.Extend(3)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("l2 extend success")
	}
	res, err = l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("left seconds: %d\n", res)

	ok, err = l1.Extend(1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("extend fail")
	}

	res, err = l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("left seconds: %d\n", res)
	time.Sleep(time.Millisecond * 1500)

	res, err = l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("left seconds: %d\n", res)

	ok, err = l1.Extend(5)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("extend fail")
	}

	res, err = l1.LeftSec()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("left seconds: %d\n", res)
}

func TestValid(t *testing.T) {
	l1 := NewLock("TestValid", pool, &LockOptions{TimeoutSec: 3})

	ok, err := l1.Valid()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Log("valid expected false but got true")
	}
	err = l1.Lock()
	if err != nil {
		t.Fatal(err)
	}
	lockId := l1.GetID()
	if lockId == "" {
		t.Fatal("empty lockId")
	}
	ok, err = l1.Valid()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Log("valid expected true but got false")
	}
}
