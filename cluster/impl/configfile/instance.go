// Package configfile 基于配置文件实现的集群实例
package configfile

import "github.com/tencent-connect/botgo-plugins/cluster/base"

// Instance 实例配置
type Instance struct {
	// Name 实例名称，需要保证唯一
	Name string `yaml:"name"`
}

// GetName 获取实例名称
func (ins *Instance) GetName() string {
	return ins.Name
}

// IsValid 是否是有效实例
func (ins *Instance) IsValid() bool {
	return ins.Name != ""
}

// IsSame 是否是相同实例
func (ins *Instance) IsSame(i base.Instance) bool {
	return ins.Name == i.GetName()
}
