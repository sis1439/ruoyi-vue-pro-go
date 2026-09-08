package system

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/cache"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/utils"
)

const (
	// Redis Key 前缀：访问令牌，与 Java 保持一致
	RedisKeyOAuth2AccessToken = "oauth2_access_token:%s"

	// 默认过期时间
	DefaultAccessTokenExpireSeconds  = 30 * 24 * 3600 // 30 天
	DefaultRefreshTokenExpireSeconds = 60 * 24 * 3600 // 60 天
)

// OAuth2AccessToken 访问令牌结构，与 Java OAuth2AccessTokenDO 对齐
type OAuth2AccessToken struct {
	AccessToken  string            `json:"accessToken"`
	RefreshToken string            `json:"refreshToken"`
	UserID       int64             `json:"userId"`
	UserType     int               `json:"userType"`
	TenantID     int64             `json:"tenantId"`
	UserInfo     map[string]string `json:"userInfo"`
	ClientID     string            `json:"clientId"`
	Scopes       []string          `json:"scopes"`
	ExpiresTime  time.Time         `json:"expiresTime"`
}

// OAuth2TokenService OAuth2 Token 服务
type OAuth2TokenService struct{}

func NewOAuth2TokenService() *OAuth2TokenService {
	return &OAuth2TokenService{}
}

// CreateAccessToken 创建访问令牌（使用 JWT 格式）
func (s *OAuth2TokenService) CreateAccessToken(ctx context.Context, userId int64, userType int, tenantId int64, userInfo map[string]string) (*OAuth2AccessToken, error) {
	// 1. 计算过期时间
	expireDuration := time.Duration(DefaultAccessTokenExpireSeconds) * time.Second
	refreshDuration := time.Duration(DefaultRefreshTokenExpireSeconds) * time.Second
	expiresTime := time.Now().Add(expireDuration)

	// 2. 获取昵称
	nickname := ""
	if userInfo != nil {
		nickname = userInfo["nickname"]
	}

	// 3. 使用 JWT 生成令牌（包含完整用户信息）
	accessToken, err := utils.GenerateTokenWithInfo(userId, userType, tenantId, nickname, expireDuration)
	if err != nil {
		return nil, err
	}
	refreshToken, err := utils.GenerateTypedToken(userId, userType, tenantId, nickname, utils.TokenRefresh, refreshDuration)
	if err != nil {
		return nil, err
	}

	// 4. 构建令牌对象
	tokenDO := &OAuth2AccessToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       userId,
		UserType:     userType,
		TenantID:     tenantId,
		UserInfo:     userInfo,
		ClientID:     "default",
		Scopes:       []string{},
		ExpiresTime:  expiresTime,
	}

	// 5. 存储到 Redis（白名单机制）
	if err := s.setAccessTokenToRedis(ctx, tokenDO); err != nil {
		return nil, err
	}

	return tokenDO, nil
}

// GetAccessToken 获取访问令牌
func (s *OAuth2TokenService) GetAccessToken(ctx context.Context, token string) (*OAuth2AccessToken, error) {
	return s.getToken(ctx, token, utils.TokenAccess)
}
func (s *OAuth2TokenService) GetRefreshToken(ctx context.Context, token string, userType int) (*OAuth2AccessToken, error) {
	result, err := s.getToken(ctx, token, utils.TokenRefresh)
	if err != nil {
		return nil, err
	}
	if result.UserType != userType {
		return nil, errors.NewBizError(401, "用户类型不匹配")
	}
	return result, nil
}
func (s *OAuth2TokenService) getToken(ctx context.Context, token, kind string) (*OAuth2AccessToken, error) {
	claims, err := utils.ParseTypedToken(token, kind)
	if err != nil {
		return nil, err
	}
	if tenant, ok := pkgContext.TenantID(ctx); ok && tenant != claims.TenantID {
		return nil, errors.NewBizError(403, "租户不匹配")
	}
	if cache.RDB == nil {
		return nil, errors.NewBizError(503, "认证服务不可用")
	}
	data, err := cache.RDB.Get(ctx, tokenKey(token, kind)).Bytes()
	if err != nil {
		return nil, err
	}
	var result OAuth2AccessToken
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	stored := result.AccessToken
	if kind == utils.TokenRefresh {
		stored = result.RefreshToken
	}
	if stored != token || result.UserID != claims.UserID || result.UserType != claims.UserType || result.TenantID != claims.TenantID {
		return nil, errors.NewBizError(401, "令牌无效")
	}
	return &result, nil
}
func tokenKey(token, kind string) string {
	if kind == utils.TokenRefresh {
		return "oauth2_refresh_token:" + token
	}
	return fmt.Sprintf(RedisKeyOAuth2AccessToken, token)
}
func (s *OAuth2TokenService) CheckAccessToken(ctx context.Context, token string) (*OAuth2AccessToken, error) {
	return s.GetAccessToken(ctx, token)
}
func (s *OAuth2TokenService) RemoveAccessToken(ctx context.Context, token string) (*OAuth2AccessToken, error) {
	record, err := s.GetAccessToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err = cache.RDB.Del(ctx, tokenKey(record.AccessToken, utils.TokenAccess), tokenKey(record.RefreshToken, utils.TokenRefresh)).Err(); err != nil {
		return nil, err
	}
	return record, nil
}

// RefreshAccessToken atomically consumes the refresh credential and revokes its access token.
// If new issuance fails, the session remains revoked (fail closed).
func (s *OAuth2TokenService) RefreshAccessToken(ctx context.Context, token string, userId int64, userType int, tenantId int64, userInfo map[string]string) (*OAuth2AccessToken, error) {
	old, err := s.GetRefreshToken(ctx, token, userType)
	if err != nil {
		return nil, err
	}
	if old.UserID != userId || old.TenantID != tenantId {
		return nil, errors.NewBizError(401, "身份不匹配")
	}
	n, err := cache.RDB.Eval(ctx, `if redis.call("EXISTS",KEYS[1]) == 0 then return 0 end; redis.call("DEL",KEYS[1],KEYS[2]); return 1`, []string{tokenKey(token, utils.TokenRefresh), tokenKey(old.AccessToken, utils.TokenAccess)}).Int()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, errors.NewBizError(401, "刷新令牌已使用或撤销")
	}
	return s.CreateAccessToken(ctx, userId, userType, tenantId, userInfo)
}
func (s *OAuth2TokenService) setAccessTokenToRedis(ctx context.Context, record *OAuth2AccessToken) error {
	if cache.RDB == nil {
		return errors.NewBizError(503, "认证服务不可用")
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	access, err := utils.ParseToken(record.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := utils.ParseTypedToken(record.RefreshToken, utils.TokenRefresh)
	if err != nil {
		return err
	}
	// Both whitelist entries become visible in one atomic Redis operation.
	return cache.RDB.Eval(ctx, `redis.call("SET",KEYS[1],ARGV[1],"PX",ARGV[2]); redis.call("SET",KEYS[2],ARGV[1],"PX",ARGV[3]); return 1`, []string{tokenKey(record.AccessToken, utils.TokenAccess), tokenKey(record.RefreshToken, utils.TokenRefresh)}, string(data), time.Until(access.ExpiresAt.Time).Milliseconds(), time.Until(refresh.ExpiresAt.Time).Milliseconds()).Err()
}
