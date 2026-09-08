package client

import "testing"

type stubClient struct {
	PayClient
	id   int64
	code string
}

func (c *stubClient) GetID() int64 { return c.id }
func (c *stubClient) Init() error  { return nil }

// TestCreatorReceivesChannelCode 锁定 T05 的根因修复：
// 工厂必须把真实渠道编码透传给构造函数，否则渠道实现会拿到 wx_unknown 并在下单时走 default 分支。
func TestCreatorReceivesChannelCode(t *testing.T) {
	var got string
	RegisterCreator("test_chan", func(channelID int64, channelCode, config string) (PayClient, error) {
		got = channelCode
		return &stubClient{id: channelID, code: channelCode}, nil
	})

	f := NewPayClientFactory()
	c, err := f.CreateOrUpdatePayClient(7, "test_chan", "{}")
	if err != nil {
		t.Fatalf("创建客户端失败: %v", err)
	}
	if got != "test_chan" {
		t.Fatalf("渠道编码未透传: got %q, want %q", got, "test_chan")
	}
	if c.GetID() != 7 {
		t.Fatalf("渠道编号错误: got %d, want 7", c.GetID())
	}
	if f.GetPayClient(7) == nil {
		t.Fatal("创建后应可从缓存取回")
	}

	f.RemovePayClient(7)
	if f.GetPayClient(7) != nil {
		t.Fatal("移除后缓存应为空")
	}
}

// TestUnknownChannelFailsLoudly 未注册渠道必须报错，不得回退到 Mock 伪造支付成功。
func TestUnknownChannelFailsLoudly(t *testing.T) {
	f := NewPayClientFactory()
	if _, err := f.CreateOrUpdatePayClient(1, "no_such_channel", "{}"); err == nil {
		t.Fatal("未注册渠道应返回错误")
	}
}
