package debug

import (
	"context"
	"fmt"
	"github.com/wxlbd/ruoyi-mall-go/internal/model"
	"github.com/wxlbd/ruoyi-mall-go/internal/service/system/sms/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"strings"
	"testing"
)

func TestDebugSMSDoesNotLogSecrets(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()
	sms, err := NewSmsClient(&model.SystemSmsChannel{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = sms.SendSms(context.Background(), "13812345678", "provider-token-fixture", []client.KeyValue{{Key: "code", Value: "739281"}})
	if err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 1 {
		t.Fatal("expected safe event")
	}
	text := fmt.Sprint(logs.All())
	for _, secret := range []string{"13812345678", "provider-token-fixture", "739281"} {
		if strings.Contains(text, secret) {
			t.Fatal("debug log leaked secret")
		}
	}
}
