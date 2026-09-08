package model

// SystemMailAccount 邮箱账号
type SystemMailAccount struct {
	ID             int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Mail           string  `gorm:"column:mail;not null;comment:邮箱" json:"mail"`
	Username       string  `gorm:"column:username;not null;comment:用户名" json:"username"`
	Password       string  `gorm:"column:password;not null;comment:密码" json:"password"`
	Host           string  `gorm:"column:host;not null;comment:SMTP服务器域名" json:"host"`
	Port           int     `gorm:"column:port;not null;comment:SMTP服务器端口" json:"port"`
	SslEnable      BitBool `gorm:"column:ssl_enable;not null;default:(0);comment:是否开启SSL" json:"sslEnable"`
	StarttlsEnable BitBool `gorm:"column:starttls_enable;not null;default:(0);comment:是否开启STARTTLS" json:"starttlsEnable"`
	TenantBaseDO
}

func (SystemMailAccount) TableName() string {
	return "system_mail_account"
}
