// Package base 集群实例接口定义
package base

// Instance 集群实例接口
type Instance interface {
	// GetID 获取实例ID，该ID应该在集群内唯一
	GetID() string
	// IsValid 是否是有效实例
	IsValid() bool
	// IsSame() 是否是同一个实例
	IsSame(ins Instance) bool
}
