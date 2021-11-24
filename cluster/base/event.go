// Package base 集群实例变化事件接口定义
package base

// EventType 事件类型
type EventType int

// WatchChan 事件监听channel
type WatchChan <-chan *WatchResponse

const (
	// EventTypeUnknown 未知类型
	EventTypeUnknown EventType = 0
	// EventTypeInsChanged 实例列表发生变化
	EventTypeInsChanged EventType = 1
)

// Event 事件接口
type Event interface {
	GetType() EventType
}

// WatchResponse watch响应
type WatchResponse struct {
	Events []Event
	Err    error
}

// NewWatchRsp 创建一个Watch响应
func NewWatchRsp(eventType EventType) *WatchResponse {
	e := &dftEvent{
		eventType: eventType,
	}
	return &WatchResponse{
		Events: []Event{e},
	}
}

// dftEvent 默认事件结构体
type dftEvent struct {
	eventType EventType
}

// GetType 获取事件类型
func (e *dftEvent) GetType() EventType {
	return e.eventType
}
