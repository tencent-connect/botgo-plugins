package etcd

import (
	"reflect"
	"testing"

	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

func TestEvent_GetType(t *testing.T) {
	type fields struct {
		eventType base.EventType
	}
	tests := []struct {
		name   string
		fields fields
		want   base.EventType
	}{
		{name: "c1", fields: fields{eventType: base.EventTypeInsChanged}, want: base.EventTypeInsChanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{
				eventType: tt.fields.eventType,
			}
			if got := e.GetType(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Event.GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}
