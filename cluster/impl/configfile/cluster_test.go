// Package configfile 基于配置文件实现的集群管理器
// 需要将服务器ip配置到文件中，只有匹配ip的实例才能注册
// 拉取实例列表时，将返回配置文件中的ip列表
package configfile

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

var (
	testCluster, _ = New("./cluster_config.yaml")
	testCtx        = context.Background()
)

func TestNew(t *testing.T) {
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "c1", args: args{filePath: ""}, wantErr: true},
		{name: "c2", args: args{filePath: "./cluster_config1.yaml"}, wantErr: true},
		{name: "c3", args: args{filePath: "./cluster_config.yaml"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.args.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCluster_RegInstance(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: true},
		{name: "c2", wantErr: true},
		{name: "c3", wantErr: false},
	}
	defer gomonkey.ApplyFuncSeq(base.GetLocalIP, []gomonkey.OutputCell{
		{Values: gomonkey.Params{"", errors.New("mock err")}, Times: 1},
		{Values: gomonkey.Params{"fakeIP", nil}, Times: 1},
		{Values: gomonkey.Params{"192.168.0.2", nil}, Times: 1},
	}).Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testCluster.RegInstance(testCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.RegInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
