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

type AppMemberSignInRecordHandler struct {
	svc *memberSvc.MemberSignInRecordService
}

func NewAppMemberSignInRecordHandler(svc *memberSvc.MemberSignInRecordService) *AppMemberSignInRecordHandler {
	return &AppMemberSignInRecordHandler{svc: svc}
}

// GetSignInRecordSummary 获得个人签到统计
func (h *AppMemberSignInRecordHandler) GetSignInRecordSummary(c *gin.Context) {
	userId := c.GetInt64(context.CtxUserIDKey)
	summary, err := h.svc.GetSignInRecordSummary(c, userId)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}
	response.WriteSuccess(c, summary)
}

// CreateSignInRecord 创建签到记录
func (h *AppMemberSignInRecordHandler) CreateSignInRecord(c *gin.Context) {
	userId := c.GetInt64(context.CtxUserIDKey)
	record, err := h.svc.CreateSignInRecord(c, userId)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	// Convert record to resp (simplified, or just return record)
	// Java doesn't return much? It returns the full record.
	// We'll return simplified App resp.
	response.WriteSuccess(c, member.AppMemberSignInRecordResp{
		ID:         record.ID,
		Day:        record.Day,
		Point:      record.Point,
		Experience: record.Experience,
		CreateTime: types.ToJsonDateTime(record.CreateTime),
	})
}

// GetSignInRecordPage 获得个人签到分页
func (h *AppMemberSignInRecordHandler) GetSignInRecordPage(c *gin.Context) {
	var r member.AppMemberSignInRecordPageReq
	if err := c.ShouldBindQuery(&r); err != nil {
		response.WriteBizError(c, errors.ErrParam)
		return
	}
	userId := c.GetInt64(context.CtxUserIDKey)

	// Reuse service method but fill UserID
	pageReq := member.MemberSignInRecordPageReq{
		PageParam: pagination.PageParam{
			PageNo:   r.PageNo,
			PageSize: r.PageSize,
		},
		UserID: userId,
	}

	pageResult, err := h.svc.GetSignInRecordPage(c, &pageReq)
	if err != nil {
		response.WriteBizError(c, err)
		return
	}

	respList := lo.Map(pageResult.List, func(item *memberModel.MemberSignInRecord, _ int) member.AppMemberSignInRecordResp {
		return member.AppMemberSignInRecordResp{
			ID:         item.ID,
			Day:        item.Day,
			Point:      item.Point,
			Experience: item.Experience,
			CreateTime: types.ToJsonDateTime(item.CreateTime),
		}
	})
	response.WriteSuccess(c, pagination.NewPageResult(respList, pageResult.Total))
}
