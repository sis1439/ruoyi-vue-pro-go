package job

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
	"testing"
)

type expireCombinationFake struct {
	promotion.CombinationRecordService
	called bool
	err    error
}

func (f *expireCombinationFake) ExpireCombinationRecord(context.Context) error {
	f.called = true
	return f.err
}
func TestCombinationRecordExpireJob(t *testing.T) {
	f := &expireCombinationFake{}
	j := NewCombinationRecordExpireJob(f)
	require.Equal(t, "combinationRecordExpireJob", j.GetHandlerName())
	require.NoError(t, j.Execute(context.Background(), ""))
	require.True(t, f.called)
	f.err = errors.New("refund failed")
	require.ErrorIs(t, j.Execute(context.Background(), ""), f.err)
}
