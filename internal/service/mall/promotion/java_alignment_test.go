package promotion

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/consts"
	model "github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"testing"
	"time"
)

type expiryTradeStub struct {
	called map[int64]int
	fail   int64
}

func (s *expiryTradeStub) CancelPaidOrder(ctx context.Context, user, id int64, kind int) error {
	s.called[id]++
	if id == s.fail {
		return errors.New("injected cancellation failure")
	}
	return nil
}
func TestJavaExpireCombinationBatchPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	q := query.Use(db)
	ctx := tenant.WithTenant(context.Background(), 1)
	activity := &model.PromotionCombinationActivity{Status: consts.CommonStatusDisable}
	require.NoError(t, db.WithContext(ctx).Create(activity).Error)
	records := []*model.PromotionCombinationRecord{}
	for _, orderID := range []int64{101, 102, 103} {
		r := &model.PromotionCombinationRecord{OrderID: orderID, UserID: 1, ActivityID: activity.ID, Status: consts.PromotionCombinationRecordStatusInProgress, ExpireTime: time.Now().Add(-time.Hour)}
		require.NoError(t, db.WithContext(ctx).Create(r).Error)
		records = append(records, r)
	}
	activities := NewCombinationActivityService(q, nil, nil)
	require.NoError(t, activities.DeleteCombinationActivity(ctx, activity.ID))
	trade := &expiryTradeStub{called: map[int64]int{}, fail: 102}
	svc := &combinationRecordService{q: q, activitySvc: activities, tradeSvc: trade}
	require.ErrorContains(t, svc.ExpireCombinationRecord(ctx), "injected cancellation failure")
	for _, r := range records {
		require.Equal(t, 1, trade.called[r.OrderID])
		require.NoError(t, db.WithContext(ctx).First(r, r.ID).Error)
		if r.OrderID == 102 {
			require.Equal(t, consts.PromotionCombinationRecordStatusInProgress, r.Status)
		} else {
			require.Equal(t, consts.PromotionCombinationRecordStatusFailed, r.Status)
		}
	}
	trade.fail = 0
	require.NoError(t, svc.ExpireCombinationRecord(ctx))
	require.Equal(t, 1, trade.called[101])
	require.Equal(t, 2, trade.called[102])
	require.Equal(t, 1, trade.called[103])
}
