package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wxlbd/ruoyi-mall-go/pkg/config"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	cfg := config.C.Database
	if cfg.Driver == "" && config.C.MySQL.DSN != "" {
		cfg.Driver = "mysql"
		cfg.DSN = config.C.MySQL.DSN
		cfg.MaxIdle = config.C.MySQL.MaxIdle
		cfg.MaxOpen = config.C.MySQL.MaxOpen
		cfg.MaxLifetime = config.C.MySQL.MaxLifetime
	}
	var dialector gorm.Dialector
	switch cfg.Driver {
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	default:
		panic("database.driver must be postgres or mysql")
	}

	// 自定义 GORM Logger，使用 Zap
	newLogger := gormlogger.New(
		ZapGormWriter{},
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true, // 忽略 RecordNotFound 错误日志 (因为我们会手动处理)
			Colorful:                  true,
			ParameterizedQueries:      true,
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		logger.Log.Fatal("failed to connect database", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Log.Fatal("failed to obtain database pool", zap.Error(err))
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)
	// 注册 AuditPlugin 自动填充 Creator、Updater、TenantID
	if err := db.Use(&AuditPlugin{}); err != nil {
		logger.Log.Fatal("failed to register AuditPlugin", zap.Error(err))
	}

	if err := db.Use(&TenantPlugin{}); err != nil {
		logger.Log.Fatal("failed to register TenantPlugin", zap.Error(err))
	}
	DB = db
	logger.Info("Database connected successfully")
	return db
}

// ZapGormWriter 适配 GORM Logger 接口
type ZapGormWriter struct{}

func (w ZapGormWriter) Printf(message string, data ...interface{}) {
	logger.Info(fmt.Sprintf(message, data...))
}

// AuditPlugin GORM 审计插件，自动填充 Creator、Updater、TenantID 字段
type AuditPlugin struct{}

// Name 插件名称
func (p *AuditPlugin) Name() string {
	return "AuditPlugin"
}

// Initialize 初始化插件，注册 Hook
func (p *AuditPlugin) Initialize(db *gorm.DB) error {
	// BeforeCreate Hook: 设置 Creator 和 TenantID
	db.Callback().Create().Before("gorm:create").Register("audit:before_create", beforeCreate)
	// BeforeUpdate Hook: 设置 Updater
	db.Callback().Update().Before("gorm:update").Register("audit:before_update", beforeUpdate)
	return nil
}

// beforeCreate 创建前的 Hook，设置 Creator 和 TenantID
func beforeCreate(db *gorm.DB) {
	// 1. 从 context 获取 gin.Context
	user := pkgContext.GetLoginUserFromContext(db.Statement.Context)
	if user == nil {
		return // 未登录，跳过
	}

	// 3. 检查并设置 Creator（string 类型）
	if hasField(db, "Creator") {
		creatorValue := strconv.FormatInt(user.UserID, 10)
		db.Statement.SetColumn("Creator", creatorValue)
	}

	// 4. 检查并设置 TenantID（int64 类型）
	// 注意：只有表中有 tenant_id 字段时才设置
	if hasField(db, "TenantID") {
		db.Statement.SetColumn("TenantID", user.TenantID)
	}
}

// beforeUpdate 更新前的 Hook，设置 Updater
func beforeUpdate(db *gorm.DB) {
	// 1. 从 context 获取 gin.Context
	user := pkgContext.GetLoginUserFromContext(db.Statement.Context)
	if user == nil {
		return // 未登录，跳过
	}

	// 3. 检查并设置 Updater（string 类型）
	if hasField(db, "Updater") {
		updaterValue := strconv.FormatInt(user.UserID, 10)
		db.Statement.SetColumn("Updater", updaterValue)
	}
}

// extractGinContext 从 context.Context 中提取 gin.Context
func extractGinContext(ctx context.Context) *gin.Context {
	if ctx == nil {
		return nil
	}

	// 尝试从 context 中获取 gin.Context
	if ginCtx, ok := ctx.Value(pkgContext.CtxGinContextKey).(*gin.Context); ok {
		return ginCtx
	}

	return nil
}

// hasField 检查 GORM Statement 的模型是否有指定字段
func hasField(db *gorm.DB, fieldName string) bool {
	if db.Statement == nil || db.Statement.Schema == nil {
		return false
	}

	// 通过 Schema 检查字段是否存在
	field := db.Statement.Schema.LookUpField(fieldName)
	return field != nil
}

// GetCreatorUpdater 获取 Creator/Updater 字符串（用户 ID）
// 用于 service 层手动设置时使用
func GetCreatorUpdater(c *gin.Context) string {
	user := pkgContext.GetLoginUser(c)
	if user == nil {
		return ""
	}
	return strconv.FormatInt(user.UserID, 10)
}
