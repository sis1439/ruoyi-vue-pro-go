package promotion

import (
	"time"

	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/pkg/types"
	"gorm.io/datatypes"
)

// PromotionDiyTemplate 装修模板表
type PromotionDiyTemplate struct {
	ID             int64                   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name           string                  `gorm:"column:name;type:varchar(64);not null;comment:模板名称" json:"name"`
	PreviewPicUrls types.StringListFromCSV `gorm:"column:preview_pic_urls;type:varchar(2000);comment:预览图片" json:"previewPicUrls"`
	Property       datatypes.JSON          `gorm:"column:property;type:longtext;comment:模板属性" json:"property"` // JSON
	Remark         string                  `gorm:"column:remark;type:varchar(255);comment:备注" json:"remark"`
	Used           model.BitBool           `gorm:"column:used;type:bit(1);not null;default:(0);comment:是否使用" json:"used"`
	UsedTime       *time.Time              `gorm:"column:used_time;comment:使用时间" json:"usedTime"`
	model.TenantBaseDO
}

func (PromotionDiyTemplate) TableName() string {
	return "promotion_diy_template"
}

// PromotionDiyPage 装修页面表
type PromotionDiyPage struct {
	ID             int64                   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TemplateID     int64                   `gorm:"column:template_id;type:bigint;not null;comment:模板ID" json:"templateId"`
	Name           string                  `gorm:"column:name;type:varchar(64);not null;comment:页面名称" json:"name"`
	Remark         string                  `gorm:"column:remark;type:varchar(255);comment:备注" json:"remark"`
	PreviewPicUrls types.StringListFromCSV `gorm:"column:preview_pic_urls;type:varchar(2000);comment:预览图片" json:"previewPicUrls"`
	Property       datatypes.JSON          `gorm:"column:property;type:longtext;comment:页面属性" json:"property"` // JSON
	model.TenantBaseDO
}

func (PromotionDiyPage) TableName() string {
	return "promotion_diy_page"
}
