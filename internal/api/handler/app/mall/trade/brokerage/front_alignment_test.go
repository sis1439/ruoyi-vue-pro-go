package brokerage

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/trade/brokerage"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	svc "github.com/wxlbd/ruoyi-mall-go/internal/service/mall/trade/brokerage"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	identity "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"go.uber.org/zap"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFrontRankIdentityAndTimeEncodingsPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := identity.WithTenant(context.Background(), 1)
	for i, v := range []struct {
		user  int64
		price int
		date  string
	}{{17, 100, "2026-09-03"}, {18, 200, "2026-09-03"}, {19, 50, "2026-09-03"}, {17, 1000, "2026-08-03"}} {
		r := &model.BrokerageRecord{UserID: v.user, BizID: string(rune('a' + i)), BizType: 1, Status: 1, Price: v.price}
		r.CreateTime, _ = time.Parse("2006-01-02", v.date)
		require.NoError(t, db.WithContext(ctx).Create(r).Error)
	}
	foreign := &model.BrokerageRecord{UserID: 99, BizID: "foreign", BizType: 1, Status: 1, Price: 9999}
	foreign.CreateTime = time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.WithContext(identity.WithTenant(context.Background(), 2)).Create(foreign).Error)
	record := svc.NewBrokerageRecordService(query.Use(db), zap.NewNop(), nil, nil, nil)
	h := NewAppBrokerageUserHandler(nil, record, nil, nil)
	for _, q := range []string{"times=1788220800000&times=1790812799999", "times[]=1788220800000&times[]=1790812799999", "times[0]=1788220800000&times[1]=1790812799999", "times=1788220800000,1790812799999"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/?"+q, nil)
		identity.SetLoginUser(c, &identity.LoginUser{UserID: 17, UserType: 1, TenantID: 1})
		c.Set("userId", int64(999))
		h.GetRankByPrice(c)
		var body struct {
			Code int `json:"code"`
			Data int `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Zero(t, body.Code, w.Body.String())
		require.Equal(t, 2, body.Data, q)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	h.GetRankByPrice(c)
	require.Contains(t, w.Body.String(), "时间范围无效")
}
