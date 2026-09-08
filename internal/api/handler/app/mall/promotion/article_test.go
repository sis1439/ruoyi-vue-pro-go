package promotion

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	dto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	"testing"
	"time"
)

func TestAppArticleResponse(t *testing.T) {
	v := articleResponse(&dto.ArticleRespVO{ID: 1, Title: "article", CreateTime: time.UnixMilli(1700000000123), Status: 1, Sort: 99})
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	require.Len(t, m, 10)
	require.Equal(t, float64(1700000000123), m["createTime"])
	for _, k := range []string{"sort", "status", "recommendHot", "recommendBanner"} {
		require.NotContains(t, m, k)
	}
}
