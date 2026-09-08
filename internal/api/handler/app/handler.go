package app

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/mall"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/pay"
	"github.com/wxlbd/ruoyi-mall-go/internal/api/handler/app/system"
)

type AppHandlers struct {
	Support *SupportHandler
	Mall    *mall.Handlers
	Member  *member.Handlers
	Pay     *pay.Handlers
	System  *system.Handlers
}
