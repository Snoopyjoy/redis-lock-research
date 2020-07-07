package lock

// 加锁与解锁的功能
type Ilock interface {
	// 尝试加锁一次
	TryLock() (bool, error)
	// 加锁 抢锁失败自动重试
	Lock() error
	// 释放锁
	Release() (bool, error)
	// 查询锁剩余的时间当
	// key 不存在时，返回 -2 。
	// 当 key 存在但没有设置剩余生存时间时，返回 -1 。
	LeftSec() (int64, error)
	// 重置锁剩余时间
	Extend(int64) (bool, error)
	// 锁的id
	GetID() string
	// 锁当前id和redis中的id是否一致
	Valid() (bool, error)
}
