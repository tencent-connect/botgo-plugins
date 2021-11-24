// Package main 示例代码
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tencent-connect/botgo-plugins/cluster/base"
	"github.com/tencent-connect/botgo-plugins/cluster/impl/etcd"
)

// TODO put your cluster name and etcd endpoits here.
var (
	clusterName = "foo_example_cluster"
	endpoints   = []string{
		"your_ip:your_port",
	}
)

func main() {
	fmt.Printf("start\n")
	cluster, err := etcd.New(clusterName, endpoints)
	if err != nil {
		fmt.Printf("new cluster failed. err:%v\n", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// 注册当前实例，id传空，则默认使用ip来拼接clusterName作为最终id
	ins, err := cluster.RegInstance(ctx, "")
	if err != nil {
		fmt.Printf("reg failed. err:%v\n", err)
		return
	}
	fmt.Printf("reg succ. ins:%+v\n", ins)
	// 启动监听集群事件
	startWatch(cluster)
	// 挂起等待
	waitExit(cluster)
}

func startWatch(cluster base.Cluster) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[RunBot]err:%v\n", r)
				os.Exit(-1)
			}
		}()
		wc, err := cluster.Watch(context.Background())
		if err != nil {
			fmt.Printf("watch failed. err:%v\n", err)
			return
		}
		for {
			wr := <-wc
			fmt.Printf("got event:%v\n", wr)
			if wr.Err != nil {
				fmt.Printf("got err:%v\n", wr.Err)
				return
			}
			for _, e := range wr.Events {
				fmt.Printf("got event type:%v\n", e.GetType())
				if e.GetType() == base.EventTypeInsChanged {
					getAllInstance(cluster)
				}
			}
		}
	}()
}

func getAllInstance(cluster base.Cluster) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	all, err := cluster.GetAllInstances(ctx)
	if err != nil {
		fmt.Printf("get all ins failed, err:%v\n", err)
		return
	}
	fmt.Printf("get all ins num:%v\n", len(all))
	for _, ins := range all {
		fmt.Printf("ins=====>:%v\n", ins.GetID())
	}
}

func waitExit(cluster base.Cluster) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	fmt.Println("exit")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 为了避免服务重启频繁触发集群重新调度，这里反注册也可以去掉，只依赖实例超时来删除实例
	// 这样如果实例能够快速重启，那么其他实例甚至感觉不出来集群变化，避免了频发调度
	_ = cluster.UnregInstance(ctx)
}
