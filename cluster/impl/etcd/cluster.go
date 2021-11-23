// Package etcd ETCD分布式实例集群管理器实现，各个实例以ip为标识
package etcd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/hanjm/etcd/clientv3"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
	"github.com/tencent-connect/botgo/log"
)

// Args 集群参数
type Args struct {
	// ClusterName 集群名称
	ClusterName string
	// EtcdEndPoints etcd地址
	EtcdEndPoints []string
	// EtcdTimeout etcd超时时间，默认DftEtcdTimeout
	EtcdTimeout time.Duration
	// HBInterval 心跳间隔，默认DftHBInterval
	HBInterval time.Duration
	// HBTimeoutCount 心跳超时次数，默认DftHBTimeoutCount
	HBTimeoutCount int64
	// WatchWakeInterval watch唤醒间隔，该时间后会自动触发一次TypeWake事件，业务可自行做逻辑
	WatchWakeInterval time.Duration
}

const (
	// DftEtcdTimeout 默认etcd超时时间
	DftEtcdTimeout = time.Second
	// DftHBInterval 默认心跳间隔
	DftHBInterval = time.Second * 3
	// DftHBTimeoutCount 默认心跳超时次数
	DftHBTimeoutCount = 3
	// DftWatchWakeInterval 默认Watch wake间隔
	DftWatchWakeInterval = time.Second * 60
)

// Cluster ETCD版本的集群管理器
type Cluster struct {
	// args 集群参数
	args Args
	// localInstance 本地实例，默认为nil，注册后赋值到此处
	localInstance *Instance
}

// New 创建集群管理器
func New(clusterName string, endpoints []string) (base.Cluster, error) {
	return NewWithArgs(NewArgs(clusterName, endpoints))
}

// NewWithArgs 使用参数构建
func NewWithArgs(args *Args) (base.Cluster, error) {
	if err := checkArgs(args); err != nil {
		return nil, err
	}
	return &Cluster{
		args: *args,
	}, nil
}

// NewArgs 构建默认参数
func NewArgs(clusterName string, endpoints []string) *Args {
	return &Args{
		ClusterName:       clusterName,
		EtcdEndPoints:     endpoints,
		EtcdTimeout:       DftEtcdTimeout,
		HBInterval:        DftHBInterval,
		HBTimeoutCount:    DftHBTimeoutCount,
		WatchWakeInterval: DftWatchWakeInterval,
	}
}

// RegInstance 注册实例
func (cluster *Cluster) RegInstance(ctx context.Context) (base.Instance, error) {
	if cluster.localInstance != nil {
		// 已注册，直接返回
		return cluster.localInstance, nil
	}
	// 创建实例
	ins, err := newInstance(cluster.args.ClusterName)
	if err != nil {
		return nil, err
	}
	cli, err := cluster.getClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	// 创建etcd节点
	err = putNode(ctx, cli, ins, cluster.getTTL())
	if err != nil {
		return nil, err
	}
	if err = cluster.startHeartBeat(ins); err != nil {
		return nil, err
	}
	// 保存本地实例
	cluster.localInstance = ins
	return ins, nil
}

// UnregInstance 注销实例
func (cluster *Cluster) UnregInstance(ctx context.Context) error {
	if cluster.localInstance == nil {
		return nil
	}
	cluster.localInstance.cancel()
	cli, err := cluster.getClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_ = delNode(ctx, cli, cluster.localInstance)
	cluster.localInstance = nil
	return nil
}

// GetAllInstances 获取所有实例的列表
func (cluster *Cluster) GetAllInstances(ctx context.Context) ([]base.Instance, error) {
	cli, err := cluster.getClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	rsp, err := cli.Get(ctx, cluster.args.ClusterName, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var instances []base.Instance
	for _, item := range rsp.Kvs {
		ins, err := newInstanceWithName(string(item.Key))
		if err != nil {
			continue
		}
		instances = append(instances, ins)
	}
	return instances, nil
}

// GetLocalInstance 获取本地实例
func (cluster *Cluster) GetLocalInstance(ctx context.Context) (base.Instance, error) {
	if cluster.localInstance == nil {
		return nil, errors.New("no valid local interface. plz register first")
	}
	return cluster.localInstance, nil
}

// Watch 监听集群事件
func (cluster *Cluster) Watch(ctx context.Context) (base.WatchChan, error) {
	cli, err := cluster.getClient()
	if err != nil {
		return nil, err
	}
	wc := make(chan *base.WatchResponse)
	go func() {
		defer func() {
			cli.Close()
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				buf = buf[:runtime.Stack(buf, false)]
				fmt.Printf("[WatchChanPanic]err:%v, stack:\n%s\n", r, buf)
				// 如果故障，则退出进程（通常不可能进入到这里）
				os.Exit(-1)
			}
		}()
		cluster.doWatch(ctx, cli, wc)
	}()
	return wc, nil
}

// 启动监听，并将结果转投到watchchan
func (cluster *Cluster) doWatch(ctx context.Context, cli *clientv3.Client, wc chan *base.WatchResponse) {
	ticker := time.NewTicker(cluster.args.WatchWakeInterval)
	rch := cli.Watch(ctx, cluster.args.ClusterName, clientv3.WithPrefix())
	// 启动watch时强制推送一次事件
	wc <- newWatchRsp(base.EventTypeInsChanged)
	for {
		select {
		case rsp, ok := <-rch:
			if !ok {
				time.Sleep(time.Millisecond)
				continue
			}
			for _, ev := range rsp.Events {
				if ev.Type != clientv3.EventTypeDelete && ev.Type != clientv3.EventTypePut {
					continue
				}
				wc <- newWatchRsp(base.EventTypeInsChanged)
				break
			}
		case <-ticker.C:
			wc <- newWatchRsp(base.EventTypeWatchWake)
		case <-ctx.Done():
			return
		}
	}
}

func (cluster *Cluster) startHeartBeat(ins *Instance) error {
	cli, err := cluster.getClient()
	if err != nil {
		return err
	}
	go func() {
		defer func() {
			cli.Close()
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				buf = buf[:runtime.Stack(buf, false)]
				fmt.Printf("[HeartBeatPanic]ins:%v, err:%v, stack:\n%s\n", ins.GetName(), r, buf)
				// 如果故障，则退出进程（通常不可能进入到这里）
				os.Exit(-1)
			}
		}()
		ticker := time.NewTicker(cluster.args.HBInterval)
		for {
			select {
			case <-ticker.C:
				_ = cluster.keepAlive(cli, ins)
			case <-ins.ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (cluster *Cluster) getCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), cluster.args.EtcdTimeout)
}

// keepAlive 尝试续期，如果续期失败，则等待下次心跳调度重新创建节点
func (cluster *Cluster) keepAlive(cli *clientv3.Client, ins *Instance) error {
	if !ins.IsValid() {
		return errors.New("invalid instance")
	}
	ctx, cancel := cluster.getCtx()
	defer cancel()
	if ins.leaseID != 0 {
		// 续租
		_, err := cli.KeepAliveOnce(ctx, ins.leaseID)
		if err != nil {
			log.Errorf("keep alive failed. err:%v", err)
			// 续租失败，则清除租约，等待下次keepAlive重新putNode
			ins.leaseID = 0
			return err
		}
	} else {
		// 如果没有租约，则尝试重新put设置租约
		if err := putNode(ctx, cli, ins, cluster.getTTL()); err != nil {
			log.Errorf("keep alive put node failed. err:%v", err)
			return err
		}
	}
	return nil
}

// getTTL 获取ttl秒数
func (cluster *Cluster) getTTL() int64 {
	return int64(cluster.args.HBInterval/time.Second) * cluster.args.HBTimeoutCount
}

func (cluster *Cluster) getClient() (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   cluster.args.EtcdEndPoints,
		DialTimeout: cluster.args.EtcdTimeout,
	})
}

func delNode(ctx context.Context, cli *clientv3.Client, ins *Instance) error {
	if !ins.IsValid() {
		return errors.New("invalid instance")
	}
	_, err := cli.Delete(ctx, ins.GetName())
	_, _ = cli.Revoke(ctx, ins.leaseID)
	ins.clear()
	return err
}

// putNode 写入etcd节点
func putNode(ctx context.Context, cli *clientv3.Client, ins *Instance, ttl int64) error {
	if !ins.IsValid() {
		return errors.New("invalid instance")
	}
	// 创建租约
	rsp, err := cli.Grant(ctx, ttl)
	if err != nil {
		return err
	}
	// 写入etcd节点，节点内容暂无实际意义，暂且写入1
	_, err = cli.Put(ctx, ins.GetName(), "1", clientv3.WithLease(rsp.ID))
	if err != nil {
		return err
	}
	ins.leaseID = rsp.ID
	return nil
}

func newWatchRsp(eventType base.EventType) *base.WatchResponse {
	e := &Event{
		eventType: eventType,
	}
	return &base.WatchResponse{
		Events: []base.Event{e},
	}
}

func checkArgs(args *Args) error {
	if args.ClusterName == "" {
		return errors.New("invalid cluster name")
	}
	if len(args.EtcdEndPoints) == 0 {
		return errors.New("invalid endpoints")
	}
	if args.EtcdTimeout < time.Second {
		return fmt.Errorf("invalid etcd timeout:%v", args.EtcdTimeout)
	}
	if args.HBInterval < time.Second {
		return fmt.Errorf("invalid heartbeat interval:%v", args.HBInterval)
	}
	if args.HBTimeoutCount < DftHBTimeoutCount {
		return fmt.Errorf("invalid heartbeat timeout count:%v", args.HBTimeoutCount)
	}
	if args.WatchWakeInterval < time.Second {
		return fmt.Errorf("invalid watch wake interval:%v", args.WatchWakeInterval)
	}
	return nil
}
