package lock

import (
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
)

var deleteScript = redis.NewScript(1, `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

var (
	ErrMaxTires = errors.New("max tires")
)

type lock struct {
	// 资源
	key        string
	resourceID string
	pool       *redis.Pool
	timeoutSec int64
	retryGap   time.Duration
	maxTries   int
}

type LockOptions struct {
	TimeoutSec int64
	RetryGap   time.Duration
	MaxTires   int
}

func NewLock(key string, pool *redis.Pool, options *LockOptions) Ilock {
	l := &lock{
		key:        key,
		pool:       pool,
		timeoutSec: 15,
		maxTries:   50,
		retryGap:   time.Millisecond * 50,
		resourceID: idGen(),
	}

	if options == nil {
		return l
	}
	if options.TimeoutSec > 0 {
		l.timeoutSec = options.TimeoutSec
	}
	if options.MaxTires > 0 {
		l.maxTries = options.MaxTires
	}
	if options.RetryGap > 0 {
		l.retryGap = options.RetryGap
	}
	return l
}

func (l *lock) TryLock() (bool, error) {
	conn := l.pool.Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do("SET", l.key, l.resourceID, "NX", "EX", l.timeoutSec))

	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		}
		return false, err
	}
	if reply == "OK" {
		return true, nil
	}
	return false, nil
}

func (l *lock) Lock() error {
	for i := 0; i < l.maxTries; i++ {
		if i != 0 {
			time.Sleep(l.retryGap)
		}
		ok, err := l.TryLock()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return ErrMaxTires
}

func (l *lock) Release() (bool, error) {
	conn := l.pool.Get()
	defer conn.Close()
	rst, err := redis.Int(deleteScript.Do(conn, l.key, l.resourceID))
	if err != nil {
		return false, err
	}
	if rst == 1 {
		return true, nil
	}
	return false, nil
}
