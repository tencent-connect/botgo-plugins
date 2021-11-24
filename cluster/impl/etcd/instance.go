package etcd

import (
	"context"
	"errors"

	"github.com/hanjm/etcd/clientv3"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

// Instance 实例，以id作为唯一标识
type Instance struct {
	// 实例id，需要保证唯一
	id string
	// ctx 生命周期控制ctx
	ctx context.Context
	// ctxCancel 用于反注册时销毁ctx
	ctxCancel context.CancelFunc
	// leaseID 租约id，用于keepalive
	leaseID clientv3.LeaseID
}

// newInstanceWithID 返回instance
func newInstanceWithID(id string) (*Instance, error) {
	if id == "" {
		return nil, errors.New("invalid id")
	}
	return &Instance{
		id: id,
	}, nil
}

// newInstance 创建集群实例
func newInstance(clusterName string, id string) (*Instance, error) {
	if clusterName == "" {
		return nil, errors.New("invalid cluster name")
	}
	if id == "" {
		// 如果没有指定id，则自动使用ip作为实例id
		// TODO 需要兼容容器场景，考虑使用设备id而非ip，避免ip重复
		var err error
		id, err = base.GetLocalIP()
		if err != nil {
			return nil, err
		}
	}
	ctxLocal, cancel := context.WithCancel(context.Background())
	return &Instance{
		id:        clusterName + "_" + id,
		ctx:       ctxLocal,
		ctxCancel: cancel,
	}, nil
}

// GetID 获取实例ID
func (ins *Instance) GetID() string {
	return ins.id
}

// IsValid 是否是有效实例
func (ins *Instance) IsValid() bool {
	return ins.id != ""
}

// cancel 停止
func (ins *Instance) cancel() {
	ins.ctxCancel()
}

// clear 清理ins
func (ins *Instance) clear() {
	ins.id = ""
	ins.leaseID = 0
}
