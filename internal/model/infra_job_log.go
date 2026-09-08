package model

import (
	"time"
)

// InfraJobLog 定时任务日志
type InfraJobLog struct {
	ID           int64      `gorm:"primaryKey;autoIncrement;comment:日志编号" json:"id"`
	JobID        int64      `gorm:"column:job_id;type:bigint;not null;comment:任务编号" json:"jobId"`
	HandlerName  string     `gorm:"column:handler_name;type:varchar(64);not null;comment:处理器的名字" json:"handlerName"`
	HandlerParam string     `gorm:"column:handler_param;type:varchar(255);comment:处理器的参数" json:"handlerParam"`
	ExecuteIndex int        `gorm:"column:execute_index;type:tinyint;not null;default:1;comment:第几次执行" json:"executeIndex"`
	BeginTime    time.Time  `gorm:"column:begin_time;not null;comment:开始执行时间" json:"beginTime"`
	EndTime      *time.Time `gorm:"column:end_time;comment:结束执行时间" json:"endTime"`
	Duration     *int       `gorm:"column:duration;type:int;comment:执行时长，单位：毫秒" json:"duration"`
	Status       int        `gorm:"column:status;type:tinyint;not null;comment:任务状态" json:"status"`
	Result       string     `gorm:"column:result;type:varchar(4000);comment:结果数据" json:"result"`
	TenantBaseDO
}

func (InfraJobLog) TableName() string {
	return "infra_job_log"
}
