package brokerage

import (
	"context"
	dto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/app/mall/trade"
	member "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/trade/brokerage"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"github.com/wxlbd/ruoyi-mall-go/pkg/pagination"

	"github.com/stretchr/testify/require"
	"testing"
)

func TestJavaFixedZeroCommission(t *testing.T) {
	s := &BrokerageRecordService{}
	zero := 0
	fixed := 17
	require.Zero(t, s.calculatePrice(10000, 10, &zero))
	require.Equal(t, 17, s.calculatePrice(10000, 10, &fixed))
	require.Equal(t, 1000, s.calculatePrice(10000, 10, nil))
	for _, input := range [][2]int{{100, -1}, {-100, 1}, {0, 10}, {100, 0}, {-100, -1}} {
		require.Zero(t, s.calculatePrice(input[0], input[1], nil))
	}
	require.Equal(t, 17, s.calculatePrice(-100, -1, &fixed))
}

func TestJavaChildSummaryPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenant.WithTenant(context.Background(), 1)
	for _, user := range []member.MemberUser{{ID: 2, Mobile: "13800000002", Nickname: "Alice", Avatar: "alice.png"}, {ID: 3, Mobile: "13800000003", Nickname: "Bob"}, {ID: 4, Mobile: "13800000004", Nickname: "Carol"}} {
		require.NoError(t, db.WithContext(ctx).Create(&user).Error)
	}
	for _, user := range []model.BrokerageUser{{ID: 2, BindUserID: 1}, {ID: 3, BindUserID: 2}, {ID: 4, BindUserID: 1}} {
		require.NoError(t, db.WithContext(ctx).Create(&user).Error)
	}
	for _, record := range []model.BrokerageRecord{{UserID: 2, BizID: "1", BizType: 1, Status: 1, Price: 100}, {UserID: 2, BizID: "2", BizType: 1, Status: 1, Price: 200}, {UserID: 4, BizID: "3", BizType: 1, Status: 2, Price: 999}} {
		require.NoError(t, db.WithContext(ctx).Create(&record).Error)
	}
	svc := &BrokerageUserService{q: query.Use(db)}
	req := &dto.AppBrokerageUserChildSummaryPageReqVO{PageParam: pagination.PageParam{PageNo: 1, PageSize: 1}, Level: 1, Sorting: "price", SortingOrder: "desc"}
	result, err := svc.GetBrokerageUserChildSummaryPage(ctx, req, 1)
	require.NoError(t, err)
	require.EqualValues(t, 2, result.Total)
	require.Len(t, result.List, 1)
	require.Equal(t, "Alice", result.List[0].Nickname)
	require.Equal(t, 300, result.List[0].BrokeragePrice)
	require.Equal(t, 2, result.List[0].BrokerageOrderCount)
	require.Equal(t, 1, result.List[0].BrokerageUserCount)
	req.Nickname = "Carol"
	result, err = svc.GetBrokerageUserChildSummaryPage(ctx, req, 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, result.Total)
	require.Zero(t, result.List[0].BrokeragePrice)
	other := tenant.WithTenant(context.Background(), 2)
	result, err = svc.GetBrokerageUserChildSummaryPage(other, req, 1)
	require.NoError(t, err)
	require.Empty(t, result.List)
}
