package member

import (
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/model"
)

type MemberUser struct {
	ID               int64      `gorm:"primaryKey;autoIncrement;comment:用户ID" json:"id"`
	Mobile           string     `gorm:"size:11;comment:手机" json:"mobile"`
	Password         string     `gorm:"size:100;default:'';comment:密码" json:"-"`
	Status           int32      `gorm:"default:0;comment:状态" json:"status"` // 参见 CommonStatusEnum
	RegisterIP       string     `gorm:"column:register_ip;size:32;default:'';comment:注册IP" json:"registerIp"`
	RegisterTerminal int32      `gorm:"column:register_terminal;default:0;comment:注册终端" json:"registerTerminal"` // 参见 TerminalEnum
	LoginIP          string     `gorm:"column:login_ip;size:32;default:'';comment:最后登录IP" json:"loginIp"`
	LoginDate        *time.Time `gorm:"column:login_date;comment:最后登录时间" json:"loginDate"`

	Email    string     `gorm:"size:255;default:'';comment:邮箱" json:"email"`
	Nickname string     `gorm:"size:30;default:'';comment:用户昵称" json:"nickname"`
	Avatar   string     `gorm:"size:255;default:'';comment:头像" json:"avatar"`
	Name     string     `gorm:"size:30;default:'';comment:真实姓名" json:"name"`
	Sex      int32      `gorm:"default:0;comment:性别" json:"sex"` // 参见 SexEnum
	Birthday *time.Time `gorm:"comment:出生日期" json:"birthday"`
	AreaID   int32      `gorm:"column:area_id;comment:所在地" json:"areaId"`
	Mark     string     `gorm:"size:255;default:'';comment:备注" json:"mark"`

	Point      int32                `gorm:"default:0;comment:积分" json:"point"`
	TagIds     model.IntListFromCSV `gorm:"type:varchar(255);comment:标签编号数组" json:"tagIds"`
	LevelID    int64                `gorm:"column:level_id;comment:等级编号" json:"levelId"`
	Experience int32                `gorm:"default:0;comment:经验" json:"experience"`
	GroupID    int64                `gorm:"column:group_id;comment:分组编号" json:"groupId"`

	model.TenantBaseDO
}

func (MemberUser) TableName() string {
	return "member_user"
}
