package member

import (
	"encoding/json"
	"github.com/gin-gonic/gin/binding"
	product "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/product"
	"testing"
)

func TestRequiredZeroValues(t *testing.T) {
	for _, input := range []string{
		`{"name":"Test","mobile":"13900000001","areaId":1,"detailAddress":"Test address","defaultStatus":false}`,
		`{"name":"Test","mobile":"13900000001","areaId":1,"detailAddress":"Test address","defaultStatus":true}`,
	} {
		var req AppAddressCreateReq
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			t.Fatal(err)
		}
		if err := binding.Validator.ValidateStruct(req); err != nil {
			t.Errorf("explicit boolean rejected: %v", err)
		}
	}
	for _, input := range []string{
		`{"name":"Test","mobile":"13900000001","areaId":1,"detailAddress":"Test address"}`,
		`{"name":"Test","mobile":"13900000001","areaId":1,"detailAddress":"Test address","defaultStatus":null}`,
	} {
		var req AppAddressCreateReq
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			t.Fatal(err)
		}
		if err := binding.Validator.ValidateStruct(req); err == nil {
			t.Error("missing/null boolean accepted")
		}
	}
	var brand product.ProductBrandCreateReq
	if err := json.Unmarshal([]byte(`{"name":"Test","picUrl":"image.png","sort":0,"status":0}`), &brand); err != nil {
		t.Fatal(err)
	}
	if err := binding.Validator.ValidateStruct(brand); err != nil {
		t.Errorf("explicit zero sort rejected: %v", err)
	}
}
