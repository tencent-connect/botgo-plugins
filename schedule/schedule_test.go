// Package schedule bot集群调度器，用于根据实例数量计算分区数量和分区号，启动bot服务
package schedule

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/token"
)

var testArgs = Args{
	Cluster:       &mockCluster{},
	BotAppID:      12345,
	BotToken:      "Bot appid.token",
	Intent:        dto.IntentGuildAtMessage,
	WatchInterval: time.Minute,
}

var testScheduler = &Scheduler{
	args:          &testArgs,
	sessionCtx:    nil,
	localInstance: &mockInstance{id: "127.0.0.1"},
}

type mockCluster struct {
}

// RegInstance 注册本地实例
func (m *mockCluster) RegInstance(ctx context.Context, id string) (base.Instance, error) {
	return nil, nil
}

// UnregInstance 注销本地实例
func (m *mockCluster) UnregInstance(ctx context.Context) error {
	return nil
}

// GetLocalInstance 获取本地实例
func (m *mockCluster) GetLocalInstance(ctx context.Context) (base.Instance, error) {
	return &mockInstance{id: "127.0.0.1"}, nil
}

// GetAllInstances 获取所有实例的列表
func (m *mockCluster) GetAllInstances(ctx context.Context) ([]base.Instance, error) {
	return nil, errors.New("mock err")
}

// Watch 监听集群事件
func (m *mockCluster) Watch(ctx context.Context) (base.WatchChan, error) {
	ch := make(chan *base.WatchResponse, 2)
	ch <- &base.WatchResponse{}
	ch <- &base.WatchResponse{}
	return ch, nil
}

func TestNew(t *testing.T) {
	type args struct {
		args *Args
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "fail",
			args: args{
				args: &Args{},
			},
			wantErr: true,
		}, {
			name: "succ",
			args: args{
				args: &testArgs,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestScheduler_Start(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "succ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testScheduler.Start(); (err != nil) != tt.wantErr {
				t.Errorf("Scheduler.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduler_doSchedule(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "succ",
			wantErr: false,
		},
	}

	defer gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {
	}).ApplyMethodSeq(reflect.TypeOf(testScheduler), "IsExitSchedule", []gomonkey.OutputCell{
		{Values: gomonkey.Params{false}, Times: 1},
		{Values: gomonkey.Params{true}, Times: 10000},
	}).Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testScheduler.doSchedule(); (err != nil) != tt.wantErr {
				t.Errorf("Scheduler.doSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduler_startSessions(t *testing.T) {
	type args struct {
		si *shardInfo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "succ",
			args: args{
				si: &shardInfo{
					shardIDs: []uint32{0},
					shardNum: 1,
					ap:       &dto.WebsocketAP{Shards: 5},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testScheduler.startSessions(tt.args.si); (err != nil) != tt.wantErr {
				t.Errorf("Scheduler.startSessions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduler_calShard(t *testing.T) {
	type args struct {
		allIns []base.Instance
	}
	tests := []struct {
		name    string
		args    args
		want    *shardInfo
		wantErr bool
	}{
		{
			name: "case1", args: args{}, want: &shardInfo{}, wantErr: false,
		}, {
			name: "case2", args: args{allIns: []base.Instance{
				&mockInstance{id: ""},
				&mockInstance{id: "127.0.0.1"},
			}}, want: nil, wantErr: true,
		}, {
			name: "case3", args: args{allIns: []base.Instance{
				&mockInstance{id: "fakeip1"},
				&mockInstance{id: "fakeip2"},
				&mockInstance{id: "127.0.0.1"},
			}}, want: &shardInfo{shardIDs: []uint32{2}, shardNum: 5}, wantErr: false,
		}, {
			name: "case4", args: args{allIns: []base.Instance{
				&mockInstance{id: "127.0.0.1"},
				&mockInstance{id: "fakeip3"},
				&mockInstance{id: "fakeip4"},
			}},
			want: &shardInfo{shardIDs: []uint32{0, 3}, shardNum: 5}, wantErr: false,
		}, {
			name: "case5", args: args{allIns: []base.Instance{
				&mockInstance{id: "fakeip5"},
				&mockInstance{id: "127.0.0.1"},
				&mockInstance{id: "fakeip6"},
			}},
			want: &shardInfo{shardIDs: []uint32{1, 4}, shardNum: 5}, wantErr: false,
		}, {
			name: "case6", args: args{allIns: []base.Instance{
				&mockInstance{id: "fakeip5"},
				&mockInstance{id: "127.0.0.1"},
				&mockInstance{id: "fakeip6"},
			}},
			want: &shardInfo{shardIDs: []uint32{1}, shardNum: 2}, wantErr: false,
		}, {
			name: "case7", args: args{allIns: []base.Instance{
				&mockInstance{id: "fakeip5"},
				&mockInstance{id: "fakeip6"},
				&mockInstance{id: "127.0.0.1"},
			}},
			want: &shardInfo{shardIDs: nil, shardNum: 2}, wantErr: false,
		}, {
			name: "case8", args: args{allIns: []base.Instance{
				&mockInstance{id: "127.0.0.1"},
				&mockInstance{id: "fakeip6"},
				&mockInstance{id: "fakeip5"},
			}},
			want: &shardInfo{shardIDs: []uint32{0}, shardNum: 2}, wantErr: false,
		}, {
			name: "case8", args: args{allIns: []base.Instance{
				&mockInstance{id: "127.0.0.1"}},
			},
			want: &shardInfo{shardIDs: []uint32{0}, shardNum: 1}, wantErr: false,
		}, {
			name: "case9", args: args{allIns: []base.Instance{
				&mockInstance{id: "fakeip6"},
				&mockInstance{id: "fakeip5"},
				&mockInstance{id: "127.0.0.1"},
			}},
			want: &shardInfo{shardIDs: []uint32{2}, shardNum: 3}, wantErr: false,
		},
	}

	botToken := token.BotToken(testArgs.BotAppID, testArgs.BotToken)
	openAPI := botgo.NewOpenAPI(botToken).WithTimeout(3 * time.Second)
	defer gomonkey.ApplyMethodSeq(reflect.TypeOf(openAPI), "WS", []gomonkey.OutputCell{
		{Values: gomonkey.Params{&dto.WebsocketAP{}, nil}, Times: 1},
		{Values: gomonkey.Params{&dto.WebsocketAP{Shards: 5}, nil}, Times: 3},
		{Values: gomonkey.Params{&dto.WebsocketAP{Shards: 2}, nil}, Times: 3},
		{Values: gomonkey.Params{&dto.WebsocketAP{Shards: 1}, nil}, Times: 1},
		{Values: gomonkey.Params{&dto.WebsocketAP{Shards: 3}, nil}, Times: 1},
	}).Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testScheduler.calShard(tt.args.allIns)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scheduler.calShard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && (!reflect.DeepEqual(got.shardIDs, tt.want.shardIDs) || got.shardNum != tt.want.shardNum) {
				t.Errorf("Scheduler.calShard() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockInstance 模拟服务实例信息
type mockInstance struct {
	id string
}

// GetName 获取名称
func (m *mockInstance) GetID() string {
	return m.id
}

// IsValid 是否有效
func (m *mockInstance) IsValid() bool {
	return m.id != ""
}

func Test_shardsEqual(t *testing.T) {
	type args struct {
		a []uint32
		b []uint32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same",
			args: args{
				a: []uint32{1, 2},
				b: []uint32{1, 2},
			},
			want: true,
		}, {
			name: "not same",
			args: args{
				a: []uint32{2},
				b: []uint32{1, 2},
			},
			want: false,
		}, {
			name: "same_nil",
			args: args{
				a: nil,
				b: nil,
			},
			want: true,
		}, {
			name: "same_empty",
			args: args{
				a: []uint32{},
				b: []uint32{},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shardsEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("shardsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewArgs(t *testing.T) {
	type args struct {
		cluster  base.Cluster
		botAppID uint64
		botToken string
		intent   dto.Intent
	}
	tests := []struct {
		name string
		args args
		want *Args
	}{
		{name: "c1", args: args{}, want: &Args{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewArgs(tt.args.cluster, tt.args.botAppID, tt.args.botToken,
				tt.args.intent); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
