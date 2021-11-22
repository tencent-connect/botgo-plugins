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
	// EventTypeWatchWake Watch定时Wake事件
	EventTypeWatchWake EventType = 2
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
