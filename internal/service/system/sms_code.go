package system

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	bzErr "github.com/wxlbd/ruoyi-mall-go/pkg/errors"
)

// ========== SMS 验证码场景常量 ==========

// SmsSceneEnum 短信验证码场景枚举
type SmsSceneEnum struct {
	Scene        int32
	TemplateCode string
	Description  string
}

// SMS 验证码场景定义（对齐 Java 版本）
var (
	// 会员场景
	SmsSceneMemberLogin     = SmsSceneEnum{1, "user-sms-login", "会员用户 - 手机号登陆"}
	SmsSceneMemberUpdateMob = SmsSceneEnum{2, "user-update-mobile", "会员用户 - 修改手机"}
	SmsSceneMemberUpdatePwd = SmsSceneEnum{3, "user-update-password", "会员用户 - 修改密码"}
	SmsSceneMemberResetPwd  = SmsSceneEnum{4, "user-reset-password", "会员用户 - 忘记密码"}

	// 后台用户场景
	SmsSceneAdminLogin    = SmsSceneEnum{21, "admin-sms-login", "后台用户 - 手机号登录"}
	SmsSceneAdminRegister = SmsSceneEnum{22, "admin-sms-register", "后台用户 - 手机号注册"}
	SmsSceneAdminResetPwd = SmsSceneEnum{23, "admin-reset-password", "后台用户 - 忘记密码"}
)

// SceneMap 场景到枚举的映射
var SceneMap = map[int32]SmsSceneEnum{
	1:  SmsSceneMemberLogin,
	2:  SmsSceneMemberUpdateMob,
	3:  SmsSceneMemberUpdatePwd,
	4:  SmsSceneMemberResetPwd,
	21: SmsSceneAdminLogin,
	22: SmsSceneAdminRegister,
	23: SmsSceneAdminResetPwd,
}

// GetSceneEnum 根据 scene 获取枚举（对齐 Java 的 getCodeByScene）
func GetSceneEnum(scene int32) *SmsSceneEnum {
	if se, ok := SceneMap[scene]; ok {
		return &se
	}
	return nil
}

// ========== SMS 验证码配置常量 ==========

const (
	// SmsCodeExpire 验证码过期时间（10 分钟，对齐 Java）
	SmsCodeExpire = 10 * time.Minute

	// SmsCodeSendFrequency 发送频率限制（1 分钟，对齐 Java）
	SmsCodeSendFrequency = 1 * time.Minute

	// SmsCodeMaxPerDay 每日最大发送数量（对齐 Java）
	SmsCodeMaxPerDay = 10

	// SmsCodeCacheKeyPrefix Redis 缓存 key 前缀
	SmsCodeCacheKeyPrefix = "sms:code:"

	// SmsCodeRateLimitPrefix 发送频率限制 key 前缀
	SmsCodeRateLimitPrefix = "sms:rate:"
)

// ========== 错误码定义（对齐 Java 版本）==========

var (
	// ErrSmsCodeNotFound 验证码不存在
	ErrSmsCodeNotFound = bzErr.NewBizError(1_002_014_000, "验证码不存在")

	// ErrSmsCodeExpired 验证码已过期
	ErrSmsCodeExpired = bzErr.NewBizError(1_002_014_001, "验证码已过期")

	// ErrSmsCodeUsed 验证码已使用
	ErrSmsCodeUsed = bzErr.NewBizError(1_002_014_002, "验证码已使用")

	// ErrSmsCodeSendTooFast 短信发送过于频繁
	ErrSmsCodeSendTooFast = bzErr.NewBizError(1_002_014_005, "短信发送过于频繁，请稍后再试")

	// ErrSmsCodeExceedMaxPerDay 超过每日发送数量限制
	ErrSmsCodeExceedMaxPerDay = bzErr.NewBizError(1_002_014_004, "今日短信发送数量已达上限")

	// ErrSmsSceneInvalid 短信场景无效
	ErrSmsSceneInvalid = bzErr.NewBizError(400, "短信场景无效")
)

// ========== SMS 验证码服务 ==========

type SmsCodeService struct {
	q    *query.Query
	rdb  *redis.Client
	send func(context.Context, string, int64, string, map[string]any) (int64, error)
}

func NewSmsCodeService(q *query.Query, rdb *redis.Client, smsSendService *SmsSendService) *SmsCodeService {
	s := &SmsCodeService{q: q, rdb: rdb}
	if smsSendService != nil {
		s.send = func(ctx context.Context, mobile string, userID int64, templateCode string, params map[string]any) (int64, error) {
			template, err := smsSendService.validateSmsTemplate(ctx, templateCode)
			if err != nil {
				return 0, err
			}
			channel, err := smsSendService.validateSmsChannel(ctx, template.ChannelId)
			if err != nil {
				return 0, err
			}
			if template.Status != consts.CommonStatusEnable || channel.Status != consts.CommonStatusEnable || (channel.Code != consts.SMSChannelCodeAliyun && channel.Code != consts.SMSChannelCodeTencent) {
				return 0, fmt.Errorf("SMS delivery unavailable")
			}
			logID, err := smsSendService.SendSingleSmsToMember(ctx, mobile, userID, templateCode, params)
			if err != nil {
				return 0, err
			}
			tenant, ok := pkgContext.TenantID(ctx)
			if !ok {
				return 0, fmt.Errorf("trusted tenant required")
			}
			logs := q.SystemSmsLog
			log, err := logs.WithContext(ctx).Where(logs.ID.Eq(logID), logs.TenantID.Eq(tenant)).First()
			if err != nil {
				return 0, err
			}
			if log.SendStatus != consts.SmsSendStatusSuccess || !strings.EqualFold(log.ApiSendCode, "OK") {
				return 0, fmt.Errorf("SMS delivery not confirmed")
			}
			return logID, nil
		}
	}
	return s
}

// Reserve the phone budget atomically across scenes and invalidate any older code.
var reserveSmsCode = redis.NewScript(`
if redis.call('EXISTS',KEYS[1]) == 1 then return -1 end
local n=tonumber(redis.call('GET',KEYS[2]) or '0')
if n >= tonumber(ARGV[1]) then return -2 end
redis.call('SET',KEYS[1],ARGV[2],'PX',ARGV[3])
n=redis.call('INCR',KEYS[2])
if n == 1 then redis.call('PEXPIRE',KEYS[2],ARGV[4]) end
redis.call('DEL',KEYS[3])
return n
`)
var activateSmsCode = redis.NewScript(`
if redis.call('GET',KEYS[1]) ~= ARGV[1] then return 0 end
redis.call('SET',KEYS[2],ARGV[2],'PX',ARGV[3])
return 1
`)

// SendSmsCode 发送短信验证码（完整版本，对齐 Java）
func (s *SmsCodeService) SendSmsCode(ctx context.Context, mobile string, scene int32, createIp string) error {
	key, err := s.getCacheKey(ctx, mobile, scene)
	if err != nil {
		return err
	}
	if s.q == nil || s.rdb == nil || s.send == nil {
		return fmt.Errorf("SMS dependencies unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tenant, _ := pkgContext.TenantID(ctx)
	nonce := uuid.NewString()
	rateKey := fmt.Sprintf("%s{%d:%s}", SmsCodeRateLimitPrefix, tenant, mobile)
	dailyKey := rateKey + ":day:" + time.Now().UTC().Format("2006-01-02")
	n, err := reserveSmsCode.Run(ctx, s.rdb.WithTimeout(time.Second), []string{rateKey, dailyKey, key}, SmsCodeMaxPerDay, nonce, SmsCodeSendFrequency.Milliseconds(), (24 * time.Hour).Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if n == -1 {
		return ErrSmsCodeSendTooFast
	}
	if n == -2 {
		return ErrSmsCodeExceedMaxPerDay
	}
	value, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return err
	}
	code := fmt.Sprintf("%06d", value.Int64())
	// Redis binds the verified code to this immutable audit ID; no OTP is needed in SQL.
	record := &model.SystemSmsCode{Mobile: mobile, Code: "[OTP]", Scene: scene, TodayIndex: int32(n), CreateIp: createIp, TenantBaseDO: model.TenantBaseDO{TenantID: tenant}}
	if err = s.q.SystemSmsCode.WithContext(ctx).Create(record); err != nil {
		return err
	}
	// A code is never usable before delivery succeeds. Failure requires no best-effort deletion.
	if _, err = s.send(ctx, mobile, 0, GetSceneEnum(scene).TemplateCode, map[string]any{"code": code}); err != nil {
		return fmt.Errorf("SMS delivery failed")
	}
	active, err := activateSmsCode.Run(ctx, s.rdb.WithTimeout(time.Second), []string{rateKey, key}, nonce, fmt.Sprintf("%d:%s", record.ID, code), SmsCodeExpire.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if active != 1 {
		return fmt.Errorf("SMS delivery reservation expired")
	}
	return nil
}

// ValidateSmsCode is a preflight only. Authentication MUST call UseSmsCode.
func (s *SmsCodeService) ValidateSmsCode(ctx context.Context, mobile string, scene int32, code string) error {
	_, err := s.readCode(ctx, mobile, scene, code, false)
	return err
}

// UseSmsCode atomically consumes the attempt and fails closed on audit write failure.
func (s *SmsCodeService) UseSmsCode(ctx context.Context, mobile string, scene int32, code string, usedIp string) error {
	id, err := s.readCode(ctx, mobile, scene, code, true)
	if err != nil {
		return err
	}
	if s.q == nil {
		return fmt.Errorf("SMS database unavailable")
	}
	tenant, _ := pkgContext.TenantID(ctx)
	l := s.q.SystemSmsCode
	info, err := l.WithContext(ctx).Where(l.ID.Eq(id), l.TenantID.Eq(tenant), l.Mobile.Eq(mobile), l.Scene.Eq(scene), l.Used.Is(false)).Updates(map[string]any{"used": true, "used_time": time.Now(), "used_ip": usedIp})
	if err != nil {
		return err
	}
	if info.RowsAffected != 1 {
		return ErrSmsCodeNotFound
	}
	return nil
}

func (s *SmsCodeService) readCode(ctx context.Context, mobile string, scene int32, code string, consume bool) (int64, error) {
	key, err := s.getCacheKey(ctx, mobile, scene)
	if err != nil {
		return 0, err
	}
	if s.rdb == nil {
		return 0, fmt.Errorf("SMS cache unavailable")
	}
	if len(code) != 6 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return 0, ErrSmsCodeNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	var val string
	if consume {
		val, err = s.rdb.WithTimeout(time.Second).GetDel(ctx, key).Result()
	} else {
		val, err = s.rdb.WithTimeout(time.Second).Get(ctx, key).Result()
	}
	if errors.Is(err, redis.Nil) {
		return 0, ErrSmsCodeNotFound
	}
	if err != nil {
		return 0, err
	}
	recordID, expected, ok := strings.Cut(val, ":")
	if !ok || subtle.ConstantTimeCompare([]byte(code), []byte(expected)) != 1 {
		return 0, ErrSmsCodeNotFound
	}
	id, err := strconv.ParseInt(recordID, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrSmsCodeNotFound
	}
	return id, nil
}

func (s *SmsCodeService) getCacheKey(ctx context.Context, mobile string, scene int32) (string, error) {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return "", fmt.Errorf("trusted tenant required")
	}
	if GetSceneEnum(scene) == nil {
		return "", ErrSmsSceneInvalid
	}
	if len(mobile) != 11 || strings.IndexFunc(mobile, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return "", fmt.Errorf("invalid mobile")
	}
	return fmt.Sprintf("%s{%d:%s}:%d", SmsCodeCacheKeyPrefix, tenant, mobile, scene), nil
}
