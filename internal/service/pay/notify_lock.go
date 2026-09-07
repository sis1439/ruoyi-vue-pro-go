package pay

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	pkgcontext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
)

// PayNotifyLock 支付通知分布式锁
type PayNotifyLock struct {
	rdb *redis.Client
}

func NewPayNotifyLock(rdb *redis.Client) *PayNotifyLock {
	return &PayNotifyLock{rdb: rdb}
}

const (
	payNotifyLockKeyPrefix = "pay:notify:lock:"
	payNotifyLockTimeout   = 120 * time.Second
)

// Lock 加锁并执行函数
// 对齐 Java: PayNotifyLockRedisDAO.lock
func (l *PayNotifyLock) Lock(ctx context.Context, taskID int64, fn func() error) (result error) {
	tenant, ok := pkgcontext.TenantID(ctx)
	if !ok || l.rdb == nil {
		return errors.New("notification lock requires trusted tenant and Redis")
	}
	lockKey := fmt.Sprintf("mall:tenant:%d:%s%d", tenant, payNotifyLockKeyPrefix, taskID)
	owner := uuid.NewString()

	// 尝试获取锁
	acquired, err := l.rdb.SetNX(ctx, lockKey, owner, payNotifyLockTimeout).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		// 锁已被其他进程持有
		return fmt.Errorf("lock already held for task %d", taskID)
	}

	// 确保释放锁
	defer func() {
		release, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		result = errors.Join(result, l.rdb.Eval(release, `if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end return 0`, []string{lockKey}, owner).Err())
	}()

	// 执行业务逻辑
	return fn()
}
