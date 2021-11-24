package etcd

import (
	"context"
	"errors"

	"github.com/hanjm/etcd/clientv3"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

// Instance 实例，以name作为区分
type Instance struct {
	// 实例名称，需要保证唯一
	name string
	// ctx 生命周期控制ctx
	ctx context.Context
	// ctxCancel 用于反注册时销毁ctx
	ctxCancel context.CancelFunc
	// leaseID 租约id，用于keepalive
	leaseID clientv3.LeaseID
}

// newInstanceWithName 返回instance
func newInstanceWithName(name string) (*Instance, error) {
	if name == "" {
		return nil, errors.New("invalid name")
	}
	return &Instance{
		name: name,
	}, nil
}

// newInstance 创建集群实例
func newInstance(clusterName string, name string) (*Instance, error) {
	if clusterName == "" {
		return nil, errors.New("invalid cluster name")
	}
	if name == "" {
		// 如果没有指定名字，则自动使用ip作为实例名称
		// TODO 需要兼容容器场景，考虑使用设备id而非ip，避免ip重复
		var err error
		name, err = base.GetLocalIP()
		if err != nil {
			return nil, err
		}
	}
	ctxLocal, cancel := context.WithCancel(context.Background())
	return &Instance{
		name:      clusterName + "_" + name,
		ctx:       ctxLocal,
		ctxCancel: cancel,
	}, nil
}

// GetName 获取实例名称
func (ins *Instance) GetName() string {
	return ins.name
}

// IsValid 是否是有效实例
func (ins *Instance) IsValid() bool {
	return ins.name != ""
}

// IsSame 是否是相同实例
func (ins *Instance) IsSame(i base.Instance) bool {
	return ins.name == i.GetName()
}

// cancel 停止
func (ins *Instance) cancel() {
	ins.ctxCancel()
}

// clear 清理ins
func (ins *Instance) clear() {
	ins.name = ""
	ins.leaseID = 0
}
