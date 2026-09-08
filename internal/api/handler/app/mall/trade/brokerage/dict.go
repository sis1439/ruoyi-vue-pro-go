package brokerage

import (
	"context"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	"strconv"
	"strings"
)

type brokerageLabels map[string]string

func loadBrokerageLabels(ctx context.Context, svc *system.DictService) (brokerageLabels, error) {
	result := brokerageLabels{}
	list, err := svc.GetSimpleDictDataList(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		if strings.HasPrefix(item.DictType, "brokerage_") {
			result[item.DictType+":"+item.Value] = item.Label
		}
	}
	return result, nil
}
func (labels brokerageLabels) label(kind string, value int, fallback string) string {
	if label := labels[kind+":"+strconv.Itoa(value)]; label != "" {
		return label
	}
	return fallback
}
