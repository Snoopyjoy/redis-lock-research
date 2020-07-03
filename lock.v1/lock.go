package lock

import (
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
)

const releaseScript = `if redis.call("get",KEYS[1]) == ARGV[1] then
return redis.call("del",KEYS[1])
else
return 0
end`

var (
	ErrMaxRetry = errors.New("max retry")
)

type lock struct {
	// 资源
	key        string
	pool       *redis.Pool
	timeoutSec int64
	retryGap   time.Duration
	maxRetry   int
}

type LockOptions struct {
	TimeoutSec int64
	RetryGap   time.Duration
	MaxRetry   int
}

func NewLock(key string, pool *redis.Pool, options *LockOptions) Ilock {
	l := &lock{
		key:        key,
		pool:       pool,
		timeoutSec: 15,
		maxRetry:   50,
		retryGap:   time.Millisecond * 50,
	}

	if options == nil {
		return l
	}
	if options.TimeoutSec > 0 {
		l.timeoutSec = options.TimeoutSec
	}
	if options.MaxRetry > 0 {
		l.maxRetry = options.MaxRetry
	}
	if options.RetryGap > 0 {
		l.retryGap = options.RetryGap
	}
	return l
}

func (l *lock) TryLock() (bool, error) {
	conn := l.pool.Get()
	defer conn.Close()
	reply, err := redis.String(conn.Do("SET", l.key, 1, "NX", "EX", l.timeoutSec))

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
	ok, err := l.TryLock()
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	for i := 0; i < l.maxRetry; i++ {
		time.Sleep(l.retryGap)
		ok, err := l.TryLock()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return ErrMaxRetry
}

func (l *lock) Release() (bool, error) {
	conn := l.pool.Get()
	defer conn.Close()
	rst, err := redis.Int(conn.Do("DEL", l.key))
	if err != nil {
		return false, err
	}
	if rst == 1 {
		return true, nil
	}
	return false, nil
}
