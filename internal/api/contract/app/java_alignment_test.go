package app_test

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	trade "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	member "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	brokerage "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/trade"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJavaAppRequestContracts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, body string
		value      interface{}
		valid      bool
	}{
		{"password", `{"password":"new-pass","code":"123456"}`, &member.AppMemberUserUpdatePasswordReq{}, true},
		{"avatar only", `{"avatar":"a.png"}`, &member.AppMemberUserUpdateReq{}, true},
		{"unknown sex", `{"sex":0}`, &member.AppMemberUserUpdateReq{}, true},
		{"zero refund", `{"orderItemId":1,"refundPrice":0,"way":10,"applyReason":"reason"}`, &trade.AppAfterSaleCreateReq{}, true},
		{"missing refund", `{"orderItemId":1,"way":10,"applyReason":"reason"}`, &trade.AppAfterSaleCreateReq{}, false},
		{"negative refund", `{"orderItemId":1,"refundPrice":-1,"way":10,"applyReason":"reason"}`, &trade.AppAfterSaleCreateReq{}, false},
		{"no logistics", `{"id":1,"logisticsId":0,"logisticsNo":"none"}`, &trade.AppAfterSaleDeliveryReq{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			err := c.ShouldBindJSON(tt.value)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?statuses=10,20,30", nil)
	var req trade.AppAfterSalePageReq
	require.NoError(t, c.ShouldBindQuery(&req))
	require.Equal(t, []int{10, 20, 30}, req.Statuses)
	c.Request = httptest.NewRequest("GET", "/?nickname=test&level=2&sortingField.field=price&sortingField.order=asc", nil)
	var child brokerage.AppBrokerageUserChildSummaryPageReqVO
	require.NoError(t, c.ShouldBindQuery(&child))
	require.Equal(t, "price", child.Sorting)
	require.Equal(t, "asc", child.SortingOrder)
	require.Equal(t, 2, child.Level)
	c.Request = httptest.NewRequest("GET", "/?pointStatus=false", nil)
	var settlement trade.AppTradeOrderSettlementReq
	require.NoError(t, c.ShouldBindQuery(&settlement))
	require.Zero(t, settlement.DeliveryType)
}
func TestJavaAppTimestampAndPagination(t *testing.T) {
	data, err := json.Marshal(brokerage.AppBrokerageRecordRespVO{CreateTime: types.ToJsonDateTime(time.UnixMilli(1700000000123))})
	require.NoError(t, err)
	require.Contains(t, string(data), `"createTime":1700000000123`)
	for _, size := range []int{-1, 10, 200, 201} {
		p := pagination.PageParam{PageNo: 2, PageSize: size}
		limit := p.GetLimit()
		if size == -1 {
			require.Equal(t, -1, limit)
			require.Zero(t, p.GetOffset())
		} else {
			require.LessOrEqual(t, limit, 200)
			require.Equal(t, limit, p.GetOffset())
		}
	}
}
