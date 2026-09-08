package job

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/mall/promotion"
)

type CombinationRecordExpireJob struct {
	service promotion.CombinationRecordService
}

func NewCombinationRecordExpireJob(service promotion.CombinationRecordService) *CombinationRecordExpireJob {
	return &CombinationRecordExpireJob{service: service}
}
func (j *CombinationRecordExpireJob) Execute(ctx context.Context, param string) error {
	return j.service.ExpireCombinationRecord(ctx)
}
func (j *CombinationRecordExpireJob) GetHandlerName() string { return "combinationRecordExpireJob" }
