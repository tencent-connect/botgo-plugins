// Package etcd ETCD分布式实例集群管理器实现，各个实例以ip为标识
package etcd

import (
	"context"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/hanjm/etcd/clientv3"
	"github.com/hanjm/etcd/mvcc/mvccpb"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

var (
	testClusterName = "testClusterName"
	testEndPoints   = []string{"test"}
	testCluster, _  = New(testClusterName, testEndPoints)
	testCtx         = context.Background()
	testClientV3    = &clientv3.Client{}
)

func TestNew(t *testing.T) {
	type args struct {
		clusterName string
		endpoints   []string
	}
	tests := []struct {
		name    string
		args    args
		want    base.Cluster
		wantErr bool
	}{
		{
			name:    "c1",
			args:    args{clusterName: "", endpoints: testEndPoints},
			want:    nil,
			wantErr: true,
		}, {
			name: "c2",
			args: args{clusterName: testClusterName, endpoints: testEndPoints},
			want: &Cluster{
				args: Args{
					ClusterName:    testClusterName,
					EtcdEndPoints:  testEndPoints,
					EtcdTimeout:    DftEtcdTimeout,
					HBInterval:     DftHBInterval,
					HBTimeoutCount: DftHBTimeoutCount,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.clusterName, tt.args.endpoints)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func applyEtcd() *gomonkey.Patches {
	return gomonkey.ApplyMethodSeq(reflect.TypeOf(clientv3.NewLease(testClientV3)), "Grant", []gomonkey.OutputCell{
		{Values: gomonkey.Params{&clientv3.LeaseGrantResponse{}, nil}, Times: 10000},
	}).ApplyMethodSeq(reflect.TypeOf(clientv3.NewLease(testClientV3)), "KeepAliveOnce", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 10000},
	}).ApplyMethodSeq(reflect.TypeOf(clientv3.NewLease(testClientV3)), "Revoke", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 10000},
	}).ApplyMethodSeq(reflect.TypeOf(clientv3.NewKV(testClientV3)), "Put", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 10000},
	}).ApplyMethodSeq(reflect.TypeOf(clientv3.NewKV(testClientV3)), "Delete", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 10000},
	}).ApplyMethodSeq(reflect.TypeOf(clientv3.NewKV(testClientV3)), "Get", []gomonkey.OutputCell{
		{
			Values: gomonkey.Params{&clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{{Key: []byte(testInsID)}}}, nil},
			Times:  10000,
		},
	})
}

func TestCluster_RegInstance(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: false},
	}
	defer applyEtcd().Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testCluster.RegInstance(testCtx, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.RegInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCluster_UnregInstance(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: false},
	}
	defer applyEtcd().Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testCluster.UnregInstance(testCtx); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.UnregInstance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCluster_GetAllInstances(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: false},
	}
	defer applyEtcd().Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testCluster.GetAllInstances(testCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.GetAllInstances() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCluster_GetLocalInstance(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testCluster.GetLocalInstance(testCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.GetLocalInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCluster_keepAlive(t *testing.T) {
	testEtcdCluster, _ := testCluster.(*Cluster)
	testCli, _ := testEtcdCluster.getClient()
	testIns, _ := newInstance(testClusterName, "")
	testIns2 := *testIns
	testIns2.leaseID = 1
	type args struct {
		cli *clientv3.Client
		ins *Instance
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "c1",
			args: args{
				cli: testCli,
				ins: testIns,
			},
			wantErr: false,
		}, {
			name: "c2",
			args: args{
				cli: testCli,
				ins: &testIns2,
			},
			wantErr: false,
		},
	}

	defer applyEtcd().Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testEtcdCluster.keepAlive(tt.args.cli, tt.args.ins); (err != nil) != tt.wantErr {
				t.Errorf("Cluster.keepAlive() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCluster_Watch(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "c1", wantErr: false},
	}
	defer applyEtcd().Reset()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testCluster.Watch(testCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cluster.Watch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
