package handler

import (
	"github.com/google/wire"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin"
	adminInfra "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/infra"
	adminMall "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/mall"
	adminMember "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/member"
	adminPay "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/pay"
	adminStatistics "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/statistics"
	adminSystem "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/admin/system"

	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app"
	appMall "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/mall"
	appMember "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/member"
	appPay "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/pay"
	appSystem "github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/system"
)

var ProviderSet = wire.NewSet(
	// Admin Providers
	adminSystem.ProviderSet,
	adminInfra.ProviderSet,
	adminStatistics.ProviderSet,
	adminMall.ProviderSet,
	adminPay.ProviderSet,
	adminMember.ProviderSet,
	wire.Struct(new(admin.AdminHandlers), "*"),

	// App Providers
	app.NewSupportHandler,
	appMall.ProviderSet,
	appPay.ProviderSet,
	appMember.ProviderSet,
	appSystem.ProviderSet,
	wire.Struct(new(app.AppHandlers), "*"),
)
