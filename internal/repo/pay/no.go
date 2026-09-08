package pay

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// PayNoRedisDAO 支付序号的 Redis DAO
// 对齐 Java: cn.iocoder.yudao.module.pay.dal.redis.no.PayNoRedisDAO
type PayNoRedisDAO struct {
	rdb *redis.Client
}

// NewPayNoRedisDAO 创建 PayNoRedisDAO
func NewPayNoRedisDAO(rdb *redis.Client) *PayNoRedisDAO {
	return &PayNoRedisDAO{rdb: rdb}
}

// Generate 生成序号
// 格式: prefix + yyyyMMddHHmmss + sequence
// 对齐 Java: PayNoRedisDAO.generate(String prefix)
func (dao *PayNoRedisDAO) Generate(ctx context.Context, prefix string) (string, error) {
	// 生成前缀：prefix + 当前时间（格式：yyyyMMddHHmmss）
	noPrefix := prefix + time.Now().Format("20060102150405")
	key := "pay_no:" + noPrefix

	// 递增序号
	no, err := dao.rdb.Incr(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to increment pay no: %w", err)
	}

	// 设置过期时间（1分钟）
	dao.rdb.Expire(ctx, key, time.Minute)

	// 返回完整序号
	return fmt.Sprintf("%s%d", noPrefix, no), nil
}

// AcquireSyncSlot 支付单主动查单的频率闸门，同一支付单在 ttl 内只放行一次
func (dao *PayNoRedisDAO) AcquireSyncSlot(ctx context.Context, orderID int64, ttl time.Duration) (bool, error) {
	return dao.rdb.SetNX(ctx, fmt.Sprintf("pay_order:sync:%d", orderID), "1", ttl).Result()
}
