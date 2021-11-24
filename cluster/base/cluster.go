// Package base 分布式服务集群管理模块接口定义
package base

import (
	"context"
)

// Cluster 集群管理器接口
type Cluster interface {
	// RegInstance 注册本地实例，实例name要保证集群内唯一
	RegInstance(ctx context.Context, name string) (Instance, error)
	// UnregInstance 注销本地实例
	UnregInstance(ctx context.Context) error
	// GetLocalInstance 获取本地实例
	GetLocalInstance(ctx context.Context) (Instance, error)
	// GetAllInstances 获取所有实例的列表
	GetAllInstances(ctx context.Context) ([]Instance, error)
	// Watch 监听集群事件
	Watch(ctx context.Context) (WatchChan, error)
}
