package trade

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TradeNoRedisDAO 交易号（订单号、售后单号等）的 Redis DAO
// 对齐 Java: cn.iocoder.yudao.module.trade.dal.redis.TradeNoRedisDAO
type TradeNoRedisDAO struct {
	rdb *redis.Client
}

// NewTradeNoRedisDAO 创建 TradeNoRedisDAO
func NewTradeNoRedisDAO(rdb *redis.Client) *TradeNoRedisDAO {
	return &TradeNoRedisDAO{rdb: rdb}
}

// Generate 生成序号
// 格式: prefix + yyyyMMddHHmmss + sequence
// 对齐 Java: TradeNoRedisDAO.generate(String prefix)
func (dao *TradeNoRedisDAO) Generate(ctx context.Context, prefix string) (string, error) {
	// 生成前缀：prefix + 当前时间（格式：yyyyMMddHHmmss）
	noPrefix := prefix + time.Now().Format("20060102150405")
	key := "trade_no:" + noPrefix

	// 递增序号
	no, err := dao.rdb.Incr(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to increment trade no: %w", err)
	}

	// 设置过期时间（1分钟）
	dao.rdb.Expire(ctx, key, time.Minute)

	// 返回完整序号
	return fmt.Sprintf("%s%06d", noPrefix, no), nil
}

// GenerateOrderNo 生成订单号
// 前缀: 1 (表示订单)
func (dao *TradeNoRedisDAO) GenerateOrderNo(ctx context.Context) (string, error) {
	return dao.Generate(ctx, "1")
}

// GenerateAfterSaleNo 生成售后单号
// 前缀: 2 (表示售后)
func (dao *TradeNoRedisDAO) GenerateAfterSaleNo(ctx context.Context) (string, error) {
	return dao.Generate(ctx, "2")
}

// AcquireSyncSlot 支付状态同步的频率闸门。
// 同一订单在 ttl 内只允许一次主动查单，避免前端轮询把渠道查单接口打爆。
// 返回 true 表示本次允许同步。
func (dao *TradeNoRedisDAO) AcquireSyncSlot(ctx context.Context, orderID int64, ttl time.Duration) (bool, error) {
	key := fmt.Sprintf("trade_order:sync:%d", orderID)
	return dao.rdb.SetNX(ctx, key, "1", ttl).Result()
}
