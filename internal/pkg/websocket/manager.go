package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Session 表示一个 WebSocket 连接会话
type Session struct {
	ID       string
	Conn     *websocket.Conn
	UserID   int64
	UserType int
	TenantID int64
	mu       sync.Mutex
}

// Send 发送消息到此会话
func (s *Session) Send(message []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteMessage(websocket.TextMessage, message)
}

// SendText 发送文本消息到此会话
func (s *Session) SendText(text string) error {
	return s.Send([]byte(text))
}

// Manager 管理所有 WebSocket 会话
type Manager struct {
	sessions map[string]*Session  // sessionID -> Session
	userMap  map[int64][]*Session // userID -> Sessions (一个用户可能有多个连接)
	mu       sync.RWMutex
}

// NewManager 创建新的会话管理器
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		userMap:  make(map[int64][]*Session),
	}
}

// Add 添加会话
func (m *Manager) Add(session *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	m.userMap[session.UserID] = append(m.userMap[session.UserID], session)
}

// Remove 移除会话
func (m *Manager) Remove(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionID]
	if !ok {
		return
	}
	delete(m.sessions, sessionID)

	// 从 userMap 中移除
	sessions := m.userMap[session.UserID]
	for i, s := range sessions {
		if s.ID == sessionID {
			m.userMap[session.UserID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}
	if len(m.userMap[session.UserID]) == 0 {
		delete(m.userMap, session.UserID)
	}
}

// GetByUser 获取指定用户的所有会话
func (m *Manager) GetByUser(userID int64) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userMap[userID]
}

// GetBySession 获取指定会话
func (m *Manager) GetBySession(sessionID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

// Send 发送消息给指定用户的所有连接
func (m *Manager) Send(userID int64, message []byte) {
	sessions := m.GetByUser(userID)
	for _, session := range sessions {
		_ = session.Send(message)
	}
}

// SendToSession 发送消息给指定会话
func (m *Manager) SendToSession(sessionID string, message []byte) {
	session := m.GetBySession(sessionID)
	if session != nil {
		_ = session.Send(message)
	}
}

// Broadcast 广播消息给所有用户
func (m *Manager) Broadcast(message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		_ = session.Send(message)
	}
}

// BroadcastByUserType 广播消息给指定用户类型的所有用户
func (m *Manager) BroadcastByUserType(userType int, message []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if session.UserType == userType {
			_ = session.Send(message)
		}
	}
}

// SendToTenant sends to one identity, or broadcasts within a tenant and user type when userID is zero.
func (m *Manager) SendToTenant(tenantID, userID int64, userType int, message []byte) {
	if tenantID <= 0 {
		return
	}
	m.mu.RLock()
	recipients := make([]*Session, 0)
	for _, session := range m.sessions {
		if session.TenantID == tenantID && session.UserType == userType && (userID == 0 || session.UserID == userID) {
			recipients = append(recipients, session)
		}
	}
	m.mu.RUnlock()
	for _, session := range recipients {
		_ = session.Send(message)
	}
}
