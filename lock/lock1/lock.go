package lock1

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

const releaseScript = `if redis.call("get",KEYS[1]) == ARGV[1] then
return redis.call("del",KEYS[1])
else
return 0
end`

type lock struct {
	// 资源
	key        string
	pool       *redis.Pool
	timeoutSec int64
	retryGap   time.Duration
}

func NewLock(key string, pool *redis.Pool, timeoutSec int64) Ilock {
	l := &lock{
		key:        key,
		pool:       pool,
		timeoutSec: timeoutSec,
		retryGap:   time.Millisecond * 50,
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
	return nil
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
