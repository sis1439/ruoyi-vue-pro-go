package app_test

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	product "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	trade "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/trade"
	member "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/member"
	app "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app"
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

func TestJavaReviewValidationAndResponseFields(t *testing.T) {
	for _, tt := range []struct {
		body  string
		valid bool
	}{
		{`{"type":1,"price":0}`, true},
		{`{"type":2,"price":100,"userName":"A","userAccount":"B","bankName":""}`, true},
		{`{"type":2,"price":100,"userName":"A","userAccount":"B"}`, false},
		{`{"type":6,"price":100,"userName":" ","userAccount":"B"}`, false},
		{`{"type":5,"price":29,"userName":"A","userAccount":"B","transferChannelCode":"wx_lite"}`, false},
		{`{"type":5,"price":30,"userName":"A","userAccount":"B","transferChannelCode":"wx_lite"}`, true},
		{`{"type":5,"price":30,"userName":"A","userAccount":"B","transferChannelCode":"invalid"}`, false},
	} {
		var r brokerage.AppBrokerageWithdrawCreateReqVO
		require.NoError(t, json.Unmarshal([]byte(tt.body), &r))
		require.Equal(t, tt.valid, r.Validate() == nil, tt.body)
	}
	for _, body := range []string{`{"orderItemId":1,"content":"ok","descriptionScores":5,"benefitScores":5}`, `{"orderItemId":1,"content":"ok","descriptionScores":5,"benefitScores":5,"anonymous":false}`} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		var r trade.AppTradeOrderItemCommentCreateReq
		err := c.ShouldBindJSON(&r)
		require.Equal(t, strings.Contains(body, "anonymous"), err == nil)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"mobile":"","scene":3}`))
	c.Request.Header.Set("Content-Type", "application/json")
	var sms member.AppAuthSmsSendReq
	require.NoError(t, c.ShouldBindJSON(&sms))
	for _, path := range []string{"/", "/?level=0", "/?level=1"} {
		c.Request = httptest.NewRequest("GET", path, nil)
		var r brokerage.AppBrokerageUserChildSummaryPageReqVO
		require.Equal(t, path == "/?level=1", c.ShouldBindQuery(&r) == nil)
	}
	for _, v := range []struct {
		value any
		keys  []string
	}{
		{app.AfterSaleLogResp{}, []string{"id", "content", "createTime"}},
		{app.DeliveryExpressResp{}, []string{"id", "name"}},
	} {
		b, err := json.Marshal(v.value)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(b, &m))
		require.Len(t, m, len(v.keys))
		for _, k := range v.keys {
			require.Contains(t, m, k)
		}
	}
	b, err := json.Marshal(product.AppProductCommentResp{})
	require.NoError(t, err)
	var comment map[string]any
	require.NoError(t, json.Unmarshal(b, &comment))
	require.Contains(t, comment, "skuProperties")
	require.NotContains(t, comment, "properties")
	for _, k := range []string{"userId", "anonymous", "orderId", "orderItemId", "replyStatus", "replyUserId", "additionalContent", "additionalPicUrls", "additionalTime", "spuId", "skuId", "descriptionScores", "benefitScores"} {
		require.Contains(t, comment, k)
	}
}

func TestJavaRankTimesRequired(t *testing.T) {
	for _, path := range []string{"/", "/?times[]=1700000000000", "/?times[]=1700000000000&times[]=1700100000000", "/?times[0]=1700000000000&times[1]=1700100000000", "/?times=1700000000000&times=1700100000000"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", path, nil)
		var r brokerage.AppBrokerageUserRankPageReqVO
		r.Times = types.QueryTimeRange(c.Request.URL.Query(), "times")
		err := c.ShouldBindQuery(&r)
		require.Equal(t, strings.Contains(path, "1700100000000"), err == nil, path)
	}
}
