package lock1

// 加锁与解锁的功能
type Ilock interface {
	TryLock() (bool, error)
	Lock() error
	Release() (bool, error)
}
