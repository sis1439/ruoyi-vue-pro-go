package system

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"image"
	"image/draw"
	"image/png"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wenlng/go-captcha-assets/resources/images"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
)

// 验证码配置常量
const (
	// CaptchaKeyPrefix Redis 缓存键前缀
	CaptchaKeyPrefix = "captcha:slide:"
	// CaptchaTTL 验证码有效期
	CaptchaTTL = 5 * time.Minute
	// CaptchaValidatePadding 校验容差（像素）
	CaptchaValidatePadding = 5
	// CaptchaImageWidth 主图宽度（匹配前端 aj-captcha）
	CaptchaImageWidth = 310
	// CaptchaImageHeight 主图高度（匹配前端 aj-captcha）
	CaptchaImageHeight = 155
	// CaptchaGraphSize 拼图块尺寸（固定值，确保滑块与挖空区域一致）
	CaptchaGraphSize = 47
)

// CaptchaService 验证码服务
type CaptchaService struct {
	rdb     *redis.Client
	captcha slide.Captcha

	// once 确保资源只初始化一次
	once sync.Once
	// initErr 初始化错误
	initErr error
}

// NewCaptchaService 创建验证码服务
func NewCaptchaService(rdb *redis.Client) *CaptchaService {
	svc := &CaptchaService{
		rdb: rdb,
	}
	return svc
}

// init 初始化验证码生成器（懒加载）
func (s *CaptchaService) init() error {
	s.once.Do(func() {
		builder := slide.NewBuilder(
			slide.WithGenGraphNumber(1), // 生成 1 个拼图块
			// 配置图片尺寸以匹配前端 aj-captcha 组件
			slide.WithImageSize(option.Size{Width: CaptchaImageWidth, Height: CaptchaImageHeight}),
			// 配置拼图块尺寸（固定值，确保滑块与挖空区域大小一致）
			slide.WithRangeGraphSize(option.RangeVal{Min: CaptchaGraphSize, Max: CaptchaGraphSize}),
		)

		// 加载 go-captcha-assets 预设背景图
		bgImages, err := images.GetImages()
		if err != nil {
			s.initErr = fmt.Errorf("加载背景图失败: %w", err)
			return
		}

		// 加载 go-captcha-assets 预设拼图块
		tileGraphs, err := tiles.GetTiles()
		if err != nil {
			s.initErr = fmt.Errorf("加载拼图块失败: %w", err)
			return
		}

		// 转换 tiles 到 slide.GraphImage（不手动缩放，由库内部根据 RangeGraphSize 统一处理）
		graphImages := make([]*slide.GraphImage, 0, len(tileGraphs))
		for _, graph := range tileGraphs {
			graphImages = append(graphImages, &slide.GraphImage{
				OverlayImage: graph.OverlayImage,
				MaskImage:    graph.MaskImage,
				ShadowImage:  graph.ShadowImage,
			})
		}

		// 设置资源
		builder.SetResources(
			slide.WithBackgrounds(bgImages),
			slide.WithGraphImages(graphImages),
		)

		s.captcha = builder.Make()
	})
	return s.initErr
}

// CaptchaData 存储在 Redis 中的验证码数据
type CaptchaData struct {
	X int `json:"x"` // 正确的 X 坐标
	Y int `json:"y"` // 正确的 Y 坐标
}

// GenerateResult 生成验证码结果
type GenerateResult struct {
	Token               string // 唯一标识
	OriginalImageBase64 string // 主图 Base64
	JigsawImageBase64   string // 拼图块 Base64
}

// Generate 生成滑动验证码
func (s *CaptchaService) Generate(ctx context.Context) (*GenerateResult, error) {
	if _, ok := pkgContext.TenantID(ctx); !ok {
		return nil, fmt.Errorf("trusted tenant required")
	}
	if s.rdb == nil {
		return nil, fmt.Errorf("captcha unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	// 确保初始化
	if err := s.init(); err != nil {
		return nil, fmt.Errorf("初始化验证码生成器失败: %w", err)
	}

	// 生成验证码
	captData, err := s.captcha.Generate()
	if err != nil {
		return nil, fmt.Errorf("生成验证码失败: %w", err)
	}

	// 获取正确答案坐标
	blockData := captData.GetData()
	if blockData == nil {
		return nil, fmt.Errorf("获取验证码数据失败")
	}

	// 生成唯一标识
	token := uuid.New().String()

	// 存储答案到 Redis
	data := CaptchaData{
		X: blockData.X,
		Y: blockData.Y,
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("序列化验证码数据失败: %w", err)
	}

	key, err := captchaKey(ctx, CaptchaKeyPrefix, token)
	if err != nil {
		return nil, err
	}
	if err := s.rdb.WithTimeout(time.Second).Set(ctx, key, dataBytes, CaptchaTTL).Err(); err != nil {
		return nil, fmt.Errorf("存储验证码数据失败: %w", err)
	}

	// 主图：go-captcha 返回 JPEG 格式，需转换为 PNG 以匹配前端 data:image/png;base64, 前缀
	masterImg := captData.GetMasterImage().Get()
	var masterBuf bytes.Buffer
	if err := png.Encode(&masterBuf, masterImg); err != nil {
		return nil, fmt.Errorf("转换主图为 PNG 失败: %w", err)
	}
	masterBase64 := base64.StdEncoding.EncodeToString(masterBuf.Bytes())

	// 拼图块：需要重新组装为 aj-captcha 期望的"垂直条状"格式
	// aj-captcha 前端期望 tile 高度等于主图高度，拼图块放置在 Y 坐标位置
	tileImg := captData.GetTileImage().Get()
	tileBounds := tileImg.Bounds()

	// 创建新的透明 PNG 画布，尺寸为 tile宽度 x 主图高度
	newTile := image.NewRGBA(image.Rect(0, 0, tileBounds.Dx(), CaptchaImageHeight))

	// 将原始 tile 绘制到画布的 Y 坐标位置
	destRect := image.Rect(0, blockData.Y, tileBounds.Dx(), blockData.Y+tileBounds.Dy())
	draw.Draw(newTile, destRect, tileImg, tileBounds.Min, draw.Over)

	// 编码为 PNG Base64
	var tileBuf bytes.Buffer
	if err := png.Encode(&tileBuf, newTile); err != nil {
		return nil, fmt.Errorf("转换拼图块为 PNG 失败: %w", err)
	}
	tileBase64 := base64.StdEncoding.EncodeToString(tileBuf.Bytes())

	return &GenerateResult{
		Token:               token,
		OriginalImageBase64: masterBase64,
		JigsawImageBase64:   tileBase64,
	}, nil
}

// Verify 校验验证码
// token: 验证码唯一标识
// x, y: 用户提交的坐标
// 注意：aj-captcha 前端滑块验证只提交 X 坐标（滑动距离），Y 坐标固定为 5
// 因此只需校验 X 坐标是否在容差范围内
func (s *CaptchaService) Verify(ctx context.Context, token string, x, y int) (bool, error) {
	key, err := captchaKey(ctx, CaptchaKeyPrefix, token)
	if err != nil {
		return false, err
	}
	if s.rdb == nil {
		return false, fmt.Errorf("captcha unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// 从 Redis 获取答案
	dataBytes, err := s.rdb.WithTimeout(time.Second).GetDel(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil // 验证码不存在或已过期
		}
		return false, fmt.Errorf("获取验证码数据失败: %w", err)
	}

	// GETDEL consumes the challenge atomically, including an incorrect attempt.

	// 解析答案
	var data CaptchaData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return false, fmt.Errorf("解析验证码数据失败: %w", err)
	}

	// 只校验 X 坐标（滑块验证只需要水平方向的位置）
	// 使用容差范围判断：|用户X - 正确X| <= padding
	if x < 0 || x > CaptchaImageWidth {
		return false, nil
	}
	diff := x - data.X
	if diff < 0 {
		diff = -diff
	}
	valid := diff <= CaptchaValidatePadding
	return valid, nil
}

func captchaKey(ctx context.Context, prefix, token string) (string, error) {
	tenant, ok := pkgContext.TenantID(ctx)
	if !ok {
		return "", fmt.Errorf("trusted tenant required")
	}
	id, err := uuid.Parse(token)
	if err != nil || id.String() != token {
		return "", fmt.Errorf("invalid captcha token")
	}
	return fmt.Sprintf("%s%d:%s", prefix, tenant, token), nil
}

// Check preserves aj-captcha's token---pointJson verification contract.
func (s *CaptchaService) Check(ctx context.Context, token, pointJSON string) (bool, error) {
	var point struct {
		X *float64 `json:"x"`
		Y *float64 `json:"y"`
	}
	if len(pointJSON) > 256 || json.Unmarshal([]byte(pointJSON), &point) != nil || point.X == nil || point.Y == nil || *point.X < 0 || *point.X > CaptchaImageWidth || *point.Y < 0 || *point.Y > CaptchaImageHeight {
		return false, fmt.Errorf("invalid captcha point")
	}
	valid, err := s.Verify(ctx, token, int(*point.X), int(*point.Y))
	if err != nil || !valid {
		return valid, err
	}
	key, err := captchaKey(ctx, "captcha:verified:", token)
	if err != nil {
		return false, err
	}
	digest := sha256.Sum256([]byte(token + "---" + pointJSON))
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := s.rdb.WithTimeout(time.Second).Set(ctx, key, digest[:], 2*time.Minute).Err(); err != nil {
		return false, err
	}
	return true, nil
}

// ConsumeVerification rejects absent, altered, expired, cross-tenant or reused proofs.
// The caller decides whether captcha is optional; a supplied value must always pass.
func (s *CaptchaService) ConsumeVerification(ctx context.Context, verification string) error {
	if len(verification) > 512 {
		return fmt.Errorf("invalid captcha verification")
	}
	token, _, ok := strings.Cut(verification, "---")
	if !ok {
		return fmt.Errorf("invalid captcha verification")
	}
	key, err := captchaKey(ctx, "captcha:verified:", token)
	if err != nil {
		return err
	}
	if s.rdb == nil {
		return fmt.Errorf("captcha unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	proof, err := s.rdb.WithTimeout(time.Second).GetDel(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("captcha verification unavailable or expired")
	}
	digest := sha256.Sum256([]byte(verification))
	if subtle.ConstantTimeCompare(proof, digest[:]) != 1 {
		return fmt.Errorf("invalid captcha verification")
	}
	return nil
}
