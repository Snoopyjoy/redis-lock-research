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

var touchScript = redis.NewScript(1, `
	local val = redis.call("GET", KEYS[1])
	if val then
		if val == ARGV[1] then
			return redis.call("expire", KEYS[1], ARGV[2])
		else
			return 0
		end
	else
		local ret = redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2], "NX")
		if ret then
			return 1
		end
		return 0
	end
`)

var (
	ErrMaxTries       = errors.New("max tries")
	ErrGetLockTimeout = errors.New("get lock time out")
)

type lock struct {
	// 资源
	key            string
	resourceID     string
	pool           *redis.Pool
	timeoutSec     int64         // 锁过期时间
	retryGap       time.Duration // 重试间隔
	maxTires       int           // 最大尝试次数
	getLockTimeout time.Duration // 加锁超时时间
}

type LockOptions struct {
	TimeoutSec    int64         // 锁过期时间
	RetryGap      time.Duration // 重试间隔
	MaxTries      int           // 最大尝试次数
	GetLockTmeout time.Duration // 加锁超时时间
}

func NewLock(key string, pool *redis.Pool, options *LockOptions) Ilock {
	l := &lock{
		key:        key,
		pool:       pool,
		timeoutSec: 15,
		maxTires:   50,
		retryGap:   time.Millisecond * 50,
		resourceID: idGen(),
	}

	if options == nil {
		return l
	}
	if options.TimeoutSec > 0 {
		l.timeoutSec = options.TimeoutSec
	}
	if options.MaxTries > 0 {
		l.maxTires = options.MaxTries
	}
	if options.RetryGap > 0 {
		l.retryGap = options.RetryGap
	}
	if options.GetLockTmeout > 0 {
		l.getLockTimeout = options.GetLockTmeout
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
	until := time.Now().Add(l.getLockTimeout)
	for i := 0; i < l.maxTires; i++ {
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
		if l.getLockTimeout > 0 && time.Now().After(until) {
			return ErrGetLockTimeout
		}
	}
	return ErrMaxTries
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

func (l *lock) Extend(seconds int64) (bool, error) {
	conn := l.pool.Get()
	defer conn.Close()
	if seconds == 0 {
		seconds = l.timeoutSec
	}
	rst, err := redis.Int(touchScript.Do(conn, l.key, l.resourceID, int(seconds)))
	if err != nil {
		return false, err
	}
	if rst == 1 {
		return true, nil
	}
	return false, nil
}

func (l *lock) LeftSec() (int64, error) {
	conn := l.pool.Get()
	defer conn.Close()
	rst, err := redis.Int64(conn.Do("TTL", l.key))
	if err != nil {
		return 0, err
	}
	return rst, err
}

func (l *lock) GetID() string {
	return l.resourceID
}

func (l *lock) Valid() (bool, error) {
	conn := l.pool.Get()
	defer conn.Close()
	rst, err := redis.String(conn.Do("GET", l.key))
	if err != nil {
		if err == redis.ErrNil {
			return false, nil
		}
		return false, err
	}
	return rst == l.resourceID, nil
}
