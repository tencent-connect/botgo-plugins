// Package configfile 基于配置文件实现的集群实例
package configfile

// Instance 实例配置
type Instance struct {
	// ID 实例ID，需要保证唯一
	ID string `yaml:"id"`
}

// GetID 获取实例名称
func (ins *Instance) GetID() string {
	return ins.ID
}

// IsValid 是否是有效实例
func (ins *Instance) IsValid() bool {
	return ins.ID != ""
}
