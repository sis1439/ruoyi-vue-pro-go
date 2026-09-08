package model

// InfraJob 定时任务
type InfraJob struct {
	ID             int64  `gorm:"primaryKey;autoIncrement;comment:任务编号" json:"id"`
	Name           string `gorm:"column:name;type:varchar(32);not null;comment:任务名称" json:"name"`
	Status         int    `gorm:"column:status;type:tinyint;not null;default:0;comment:任务状态" json:"status"`
	HandlerName    string `gorm:"column:handler_name;type:varchar(64);not null;comment:处理器的名字" json:"handlerName"`
	HandlerParam   string `gorm:"column:handler_param;type:varchar(255);comment:处理器的参数" json:"handlerParam"`
	CronExpression string `gorm:"column:cron_expression;type:varchar(32);not null;comment:CRON 表达式" json:"cronExpression"`
	RetryCount     int    `gorm:"column:retry_count;type:int;not null;default:0;comment:重试次数" json:"retryCount"`
	RetryInterval  int    `gorm:"column:retry_interval;type:int;not null;default:0;comment:重试间隔，单位：毫秒" json:"retryInterval"`
	MonitorTimeout *int   `gorm:"column:monitor_timeout;type:int;comment:监控超时时间，单位：毫秒" json:"monitorTimeout"`
	TenantBaseDO
}

func (InfraJob) TableName() string {
	return "infra_job"
}
