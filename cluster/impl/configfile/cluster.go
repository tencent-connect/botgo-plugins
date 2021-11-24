// Package configfile 基于配置文件实现的集群管理器
// 需要将服务器名称配置到文件中，需要保证唯一，只有匹配名称的实例才能注册
// 拉取实例列表时，将返回配置文件中的实例列表
package configfile

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/tencent-connect/botgo-plugins/cluster/base"
	"gopkg.in/yaml.v3"
)

// ConfigFile 配置文件
type ConfigFile struct {
	InstanceList []*Instance `yaml:"instance_list"`
}

// Cluster 基于yaml配置文件的集群管理器
type Cluster struct {
	baseInsList   []base.Instance
	localInstance base.Instance
}

// New 创建集群管理器
func New(filePath string) (base.Cluster, error) {
	if filePath == "" {
		return nil, errors.New("invalid config path")
	}
	cluster := &Cluster{}
	err := cluster.loadConfig(filePath)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// RegInstance 注册本地实例，id传空则默认使用ip作为id，只有配置文件中的id才能注册成功，否则反错
func (cluster *Cluster) RegInstance(ctx context.Context, id string) (base.Instance, error) {
	if id == "" {
		var err error
		id, err = base.GetLocalIP()
		if err != nil {
			return nil, err
		}
	}
	for _, ins := range cluster.baseInsList {
		// 查找本机是否在配置的ins列表，如果在，则注册成功
		if ins.GetID() == id {
			cluster.localInstance = ins
			return ins, nil
		}
	}
	// 本机不在配置中，返回失败
	return nil, fmt.Errorf("invalid instance. id:%v", id)
}

// UnregInstance 注销本地实例
func (cluster *Cluster) UnregInstance(ctx context.Context) error {
	cluster.localInstance = nil
	return nil
}

// GetLocalInstance 获取本地实例
func (cluster *Cluster) GetLocalInstance(ctx context.Context) (base.Instance, error) {
	if cluster.localInstance == nil {
		return nil, errors.New("no instance registed")
	}
	return cluster.localInstance, nil
}

// GetAllInstances 获取所有实例的列表
func (cluster *Cluster) GetAllInstances(ctx context.Context) ([]base.Instance, error) {
	return cluster.baseInsList, nil
}

// Watch 监听集群事件
func (cluster *Cluster) Watch(ctx context.Context) (base.WatchChan, error) {
	// 创建一个buffer为1的channel，并往其中写入一个事件
	wc := make(chan *base.WatchResponse, 1)
	wc <- base.NewWatchRsp(base.EventTypeInsChanged)
	return wc, nil
}

func (cluster *Cluster) loadConfig(filePath string) error {
	cfg := &ConfigFile{}
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(buf, cfg); err != nil {
		return err
	}
	if len(cfg.InstanceList) == 0 {
		return fmt.Errorf("no instance. plz check config:%v", filePath)
	}
	for _, ins := range cfg.InstanceList {
		cluster.baseInsList = append(cluster.baseInsList, ins)
	}
	return nil
}
