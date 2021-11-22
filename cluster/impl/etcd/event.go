package etcd

import (
	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

// Event 事件
type Event struct {
	eventType base.EventType
}

// GetType 获取事件类型
func (e *Event) GetType() base.EventType {
	return e.eventType
}
