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
	"github.com/tencent-connect/botgo-plugins/schedule"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/websocket"
)

// TODO put your cluster args here
var (
	clusterName = "foo_example_cluster"
	endpoints   = []string{
		"your_ip:your_port",
	}
	botAppID = uint64(123456) // your bot appid here
	botToken = "your bot token here"
)

func main() {
	// 先注册本机实例
	cluster, err := etcd.New(clusterName, endpoints)
	if err != nil {
		fmt.Printf("new cluster failed. err:%v\n", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	ins, err := cluster.RegInstance(ctx)
	if err != nil {
		fmt.Printf("reg failed. err:%v\n", err)
		return
	}
	fmt.Printf("reg succ. ins:%+v\n", ins)
	// 创建调度器
	schedArgs := &schedule.Args{
		Cluster:  cluster,
		BotAppID: botAppID,
		BotToken: botToken,
		Intent:   websocket.RegisterHandlers(websocket.ATMessageEventHandler(msgHandler)),
	}
	sched, err := schedule.New(schedArgs)
	if err != nil {
		fmt.Printf("New sched failed. err:%v\n", err)
		return
	}
	// 启动调度
	err = sched.Start()
	if err != nil {
		fmt.Printf("Start failed. err:%v\n", err)
		return
	}
	// 挂起当前线程
	waitExit(cluster)
}

// msgHandler 消息处理
func msgHandler(event *dto.WSPayload, m *dto.WSATMessageData) error {
	fmt.Printf("receive msg:%v\n", m)
	return nil
}

func waitExit(cluster base.Cluster) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("exit")
	<-signalChan
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 为了避免服务重启频繁触发集群重新调度，这里反注册也可以去掉，只依赖实例超时来删除实例
	// 这样如果实例能够快速重启，那么其他实例甚至感觉不出来集群变化，避免了频发调度
	_ = cluster.UnregInstance(ctx)
}
