package lock

// 加锁与解锁的功能
type Ilock interface {
	// 尝试加锁一次
	TryLock() (bool, error)
	// 加锁 抢锁失败自动重试
	Lock() error
	// 释放锁
	Release() (bool, error)
}
