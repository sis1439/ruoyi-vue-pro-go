package promotion

import (
	"context"
	"encoding/json"
	"fmt"
	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	dto "github.com/wxlbd/ruoyi-mall-go/internal/api/contract/admin/mall/promotion"
	membermodel "github.com/wxlbd/ruoyi-mall-go/internal/model/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/model/promotion"
	ws "github.com/wxlbd/ruoyi-mall-go/internal/pkg/websocket"
	"github.com/wxlbd/ruoyi-mall-go/internal/repo/query"
	membersvc "github.com/wxlbd/ruoyi-mall-go/internal/service/member"
	"github.com/wxlbd/ruoyi-mall-go/internal/testutil"
	tenant "github.com/wxlbd/ruoyi-mall-go/pkg/context"
	"github.com/wxlbd/ruoyi-mall-go/pkg/database"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFrontKefuWebSocketContractAndIsolationPostgres(t *testing.T) {
	db := testutil.PostgreSQL(t)
	require.NoError(t, db.Use(&database.TenantPlugin{}))
	ctx := tenant.WithTenant(context.Background(), 1)
	q := query.Use(db)
	member := &membermodel.MemberUser{ID: 17, Mobile: "15500000017", Nickname: "Member"}
	require.NoError(t, db.WithContext(ctx).Create(member).Error)
	manager := ws.NewManager()
	accepted := make(chan *gorilla.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := (&gorilla.Upgrader{}).Upgrade(w, r, nil)
		if err == nil {
			accepted <- c
		}
	}))
	defer server.Close()
	identities := []struct {
		tenant, user int64
		kind         int
		receives     bool
	}{{1, 17, 1, true}, {1, 17, 2, true}, {1, 77, 2, true}, {1, 88, 1, false}, {2, 17, 1, false}, {2, 77, 2, false}}
	var clients []*gorilla.Conn
	var sessions []*ws.Session
	for i, v := range identities {
		client, _, err := gorilla.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		require.NoError(t, err)
		defer client.Close()
		conn := <-accepted
		defer conn.Close()
		session := &ws.Session{ID: fmt.Sprint(i), Conn: conn, TenantID: v.tenant, UserID: v.user, UserType: v.kind}
		manager.Add(session)
		clients = append(clients, client)
		sessions = append(sessions, session)
	}
	svc := NewKefuService(q, membersvc.NewMemberUserService(q, nil, nil, nil), nil, manager)
	id, err := svc.CreateMessage(ctx, dto.KefuMessageCreateReq{ContentType: 1, Content: "hello"}, 17, 1)
	require.NoError(t, err)
	var saved promotion.PromotionKefuMessage
	require.NoError(t, db.WithContext(ctx).First(&saved, id).Error)
	require.NoError(t, svc.UpdateMessageReadStatus(ctx, saved.ConversationID, 77, 2))
	// Missing member destinations and missing tenant identities must never broadcast.
	svc.(*kefuService).sendKefuMessageNotify(ctx, 0, 1, "unexpected", map[string]int{"id": 0})
	svc.(*kefuService).sendKefuMessageNotify(context.Background(), 0, 2, "unexpected", map[string]int{"id": 0})
	for _, session := range sessions {
		require.NoError(t, session.SendText("done"))
	}
	for i, client := range clients {
		require.NoError(t, client.SetReadDeadline(time.Now().Add(3*time.Second)))
		if identities[i].receives {
			for _, expected := range []string{"kefu_message_type", "kefu_message_read_status_change"} {
				_, data, err := client.ReadMessage()
				require.NoError(t, err)
				var envelope struct {
					Type    string `json:"type"`
					Content string `json:"content"`
				}
				require.NoError(t, json.Unmarshal(data, &envelope))
				require.Equal(t, expected, envelope.Type)
				var content map[string]any
				require.NoError(t, json.Unmarshal([]byte(envelope.Content), &content))
				require.Equal(t, float64(saved.ConversationID), content["conversationId"])
				if expected == "kefu_message_type" {
					require.Equal(t, "hello", content["content"])
					require.Equal(t, float64(id), content["id"])
				}
			}
		}
		_, data, err := client.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, "done", string(data), "unexpected notification to identity %v", identities[i])
	}
}
