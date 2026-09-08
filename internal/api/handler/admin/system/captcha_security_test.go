package system

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	svc "github.com/wxlbd/ruoyi-mall-go/internal/service/system"
	pkgContext "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCaptchaHTTPContract(t *testing.T) {
	addr := os.Getenv("T09_REDIS_ADDR")
	if addr == "" {
		t.Skip("T09_REDIS_ADDR required")
	}
	r := redis.NewClient(&redis.Options{Addr: addr, MaxRetries: -1})
	defer r.Close()
	if err := r.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	service := svc.NewCaptchaService(r)
	handler := NewCaptchaHandler(service)
	engine := gin.New()
	engine.POST("/get", handler.Get)
	engine.POST("/check", handler.Check)
	tenant := time.Now().UnixNano()
	ctx := pkgContext.WithTenant(context.Background(), tenant)
	request := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}
	if w := request("/get", ""); !strings.Contains(w.Body.String(), `"repCode":"0000"`) {
		t.Fatalf("empty get body compatibility: %s", w.Body)
	}
	w := request("/get", `{"captchaType":"blockPuzzle"}`)
	var got struct {
		RepCode string         `json:"repCode"`
		RepData map[string]any `json:"repData"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.RepCode != "0000" {
		t.Fatalf("get: %s", w.Body)
	}
	for field := range got.RepData {
		switch field {
		case "originalImageBase64", "jigsawImageBase64", "token", "secretKey":
		default:
			t.Fatalf("unexpected answer field %s", field)
		}
	}
	if got.RepData["secretKey"] != "" {
		t.Fatal("unexpected encryption contract")
	}
	token := got.RepData["token"].(string)
	key := fmt.Sprintf("%s%d:%s", svc.CaptchaKeyPrefix, tenant, token)
	data, err := r.Get(ctx, key).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var answer svc.CaptchaData
	if err := json.Unmarshal(data, &answer); err != nil {
		t.Fatal(err)
	}
	point := fmt.Sprintf(`{"x":%d,"y":5}`, answer.X)
	body, _ := json.Marshal(map[string]string{"token": token, "pointJson": point})
	w = request("/check", string(body))
	if !strings.Contains(w.Body.String(), `"repCode":"0000"`) {
		t.Fatalf("check: %s", w.Body)
	}
	if err := service.ConsumeVerification(ctx, token+"---"+point); err != nil {
		t.Fatal(err)
	}
	w = request("/check", string(body))
	if strings.Contains(w.Body.String(), `"repCode":"0000"`) {
		t.Fatal("HTTP replay accepted")
	}
	w = request("/get", strings.Repeat("x", 16385))
	if strings.Contains(w.Body.String(), `"repCode":"0000"`) {
		t.Fatal("oversized body accepted")
	}
	r.Close()
	w = request("/get", `{}`)
	if strings.Contains(w.Body.String(), "redis") || strings.Contains(w.Body.String(), addr) || strings.Contains(w.Body.String(), `"repCode":"0000"`) {
		t.Fatalf("outage leaked implementation detail: %s", w.Body)
	}
}
