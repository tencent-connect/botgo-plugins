package etcd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/hanjm/etcd/clientv3"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
)

// Instance 实例，以ip作为区分
type Instance struct {
	// 实例名称
	name string
	// ip 实例ip
	ip string
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
func newInstance(clusterName string) (*Instance, error) {
	if clusterName == "" {
		return nil, errors.New("invalid cluster name")
	}
	ip, err := getLocalIP()
	if err != nil {
		return nil, err
	}
	ctxLocal, cancel := context.WithCancel(context.Background())
	return &Instance{
		name:      clusterName + "_" + ip,
		ip:        ip,
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

// getLocalIP 获取本机IP
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			// 跳过loopback实例
			continue
		}
		if strings.Index(iface.Name, "eth") == 0 {
			// 返回首个eth网口的ip
			return getIP(iface)
		}
	}
	return "", fmt.Errorf("no valid iface:%v", interfaces)
}

// getIP 获取网口ip
func getIP(iface net.Interface) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, v := range addrs {
		ipNet, ok := v.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil ||
			ipNet.IP.To16() != nil {
			if ip := ipNet.IP.String(); ip != "" {
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("iface have no valid ip:%v", iface)
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
