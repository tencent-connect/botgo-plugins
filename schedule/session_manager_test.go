package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/token"
	"github.com/tencent-connect/botgo/websocket"
)

var testShardInfo = &shardInfo{
	shardIDs: []uint32{0, 1},
	shardNum: 2,
	ap: &dto.WebsocketAP{
		Shards: 2,
		SessionStartLimit: dto.SessionStartLimit{
			MaxConcurrency: 10,
		},
	},
}

var testIntent = dto.IntentGuildAtMessage

func Test_SessionManager_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	count := 0
	type fields struct {
		ctx        context.Context
		token      *token.Token
		intents    *dto.Intent
		si         *shardInfo
		holderChan chan *sessionHolder
		holders    []*sessionHolder
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				ctx:     ctx,
				si:      testShardInfo,
				token:   &token.Token{},
				intents: &testIntent,
			},
			wantErr: false,
		},
	}

	defer gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
		count++
		// 第一次循环后，则关闭session mgr
		if count == 1 {
			cancel()
		}
	}).Reset()
	websocket.Register(&MockBotWebSocket{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &SessionManager{
				ctx:        tt.fields.ctx,
				token:      tt.fields.token,
				intents:    tt.fields.intents,
				si:         tt.fields.si,
				holderChan: tt.fields.holderChan,
				holders:    tt.fields.holders,
			}
			if err := mgr.Start(); (err != nil) != tt.wantErr {
				t.Errorf("SessionManager.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// MockBotWebSocket 需要实现的接口
type MockBotWebSocket struct{}

// New 创建一个新的ws实例，需要传递 session 对象
func (m *MockBotWebSocket) New(session dto.Session) websocket.WebSocket {
	return &MockBotWebSocket{}
}

// Connect 连接到 wss 地址
func (m *MockBotWebSocket) Connect() error {
	return nil
}

// Identify 鉴权连接
func (m *MockBotWebSocket) Identify() error {
	return nil
}

// Session 拉取 session 信息，包括 token，shard，seq 等
func (m *MockBotWebSocket) Session() *dto.Session {
	return nil
}

// Resume 重连
func (m *MockBotWebSocket) Resume() error {
	return nil
}

// Listening 监听websocket事件
func (m *MockBotWebSocket) Listening() error {
	return nil
}

// Write 发送数据
func (m *MockBotWebSocket) Write(message *dto.WSPayload) error {
	return nil
}

// Close 关闭连接
func (m *MockBotWebSocket) Close() {
}
