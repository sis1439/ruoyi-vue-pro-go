package member

import (
	"github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	memberModel "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	memberSvc "github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/errors"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/response"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type AppMemberPointRecordHandler struct {
	svc *memberSvc.MemberPointRecordService
}

func NewAppMemberPointRecordHandler(svc *memberSvc.MemberPointRecordService) *AppMemberPointRecordHandler {
	return &AppMemberPointRecordHandler{svc: svc}
}

// GetPointRecordPage 获得用户积分记录分页
func (h *AppMemberPointRecordHandler) GetPointRecordPage(c *gin.Context) {
	var r member.AppMemberPointRecordPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	r.CreateTime = types.QueryTimeRange(c.Request.URL.Query(), "createTime")
	userId := context.GetLoginUserID(c)
	pageResult, err := h.svc.GetAppPointRecordPage(c, userId, &r)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	response.WriteSuccess(c, pagination.NewPageResult(lo.Map(pageResult.List, func(item *memberModel.MemberPointRecord, _ int) *member.AppMemberPointRecordResp {
		return &member.AppMemberPointRecordResp{
			ID:          item.ID,
			Title:       item.Title,
			Description: item.Description,
			Point:       item.Point,
			CreateTime:  types.ToJsonDateTime(item.CreateTime),
		}
	}), pageResult.Total))
}
