package promotion

import "github.com/wxlbd/ruoyi-mall-go/pkg/types"

type AppArticleResp struct {
	ID           int64              `json:"id"`
	Title        string             `json:"title"`
	Author       string             `json:"author"`
	CategoryID   int64              `json:"categoryId"`
	PicURL       string             `json:"picUrl"`
	Introduction string             `json:"introduction"`
	Content      string             `json:"content"`
	CreateTime   types.JsonDateTime `json:"createTime"`
	BrowseCount  int                `json:"browseCount"`
	SpuID        int64              `json:"spuId"`
}
