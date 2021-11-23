// Package schedule bot集群调度器，根据北极星服务实例数量，以及频道AP信息中的
// 最小分区数，实时计算分区数量和分区号，并启动bot服务
package schedule

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo-plugins/cluster/base"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/log"
	"github.com/tencent-connect/botgo/token"
)

// Args 调度参数
type Args struct {
	// Cluster 集群管理器
	Cluster base.Cluster
	// BotAppID Bot appid
	BotAppID uint64
	// BotToken
	BotToken string
	// Intent 注册事件
	Intent dto.Intent
}

// Scheduler 调度器对象，通过NewScheduler构造对象，提供调度接口
type Scheduler struct {
	args          Args
	localInstance base.Instance
	sessionCtx    *botSessionCtx
}

// shardInfo bot分区信息
type shardInfo struct {
	// shardIDs 需要处理的分区id列表
	shardIDs []uint
	// shardNum 分区总数
	shardNum uint
	// ap bot gateway ap信息
	ap *dto.WebsocketAP
}

// botSessions 记录bot session相关的上下文信息
type botSessionCtx struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	si         *shardInfo
	wg         sync.WaitGroup
}

// New 创建调度器对象
func New(args *Args) (*Scheduler, error) {
	// 校验参数是否合法
	if !args.isValid() {
		return nil, fmt.Errorf("invalid args %+v", args)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ins, err := args.Cluster.GetLocalInstance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get local ins failed. err:%v", err)
	}
	return &Scheduler{
		args:          *args,
		localInstance: ins,
	}, nil
}

// Start 启动调度协程监听北极星服务实例数量，并根据实例数量计算分区信息，启动对应bot session
func (sched *Scheduler) Start() error {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				buf = buf[:runtime.Stack(buf, false)]
				log.Errorf("[ScheduleMain]err:%v, stack:\n%s", r, buf)
				// 如果故障，则退出进程（通常不可能进入到这里）
				os.Exit(-1)
			}
		}()
		if err := sched.doSchedule(); err != nil {
			panic(fmt.Sprintf("do schedule failed, err:%v", err))
		}
	}()
	return nil
}

// IsExitSchedule 始终返回false，用于测试打桩构造循环退出条件
func (sched *Scheduler) IsExitSchedule() bool {
	return false
}

func (sched *Scheduler) doSchedule() error {
	wc, err := sched.args.Cluster.Watch(context.Background())
	if err != nil {
		return err
	}
	for {
		if sched.IsExitSchedule() {
			break
		}
		wr, ok := <-wc
		if !ok || wr.Err != nil {
			time.Sleep(time.Second)
			continue
		}
		if err := sched.sharding(); err != nil {
			time.Sleep(time.Second)
			continue
		}
	}

	return nil
}

func (sched *Scheduler) getTimeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second)
}

// sharding 计算分区，根据情况启动或者停止bot session
func (sched *Scheduler) sharding() error {
	ctx, cancel := sched.getTimeoutCtx()
	defer cancel()
	insList, err := sched.args.Cluster.GetAllInstances(ctx)
	if err != nil {
		log.Errorf("get all instances failed, err:%v", err)
		return err
	}

	shard, err := sched.calShard(insList)
	if err != nil {
		log.Errorf("calculate shard failed, err:%v", err)
		return err
	}

	if sched.needReschedule(shard) {
		if err := sched.reschedule(shard); err != nil {
			log.Errorf("reschedule failed, err:%v", err)
			return err
		}
	}
	return nil
}

func (sched *Scheduler) isSelf(ins base.Instance) bool {
	return sched.localInstance.IsSame(ins)
}

func (s *shardInfo) isValid() bool {
	return len(s.shardIDs) > 0 && s.shardNum > 0
}

func (s *shardInfo) isSame(si *shardInfo) bool {
	return shardsEqual(s.shardIDs, si.shardIDs) && s.shardNum == si.shardNum
}

func shardsEqual(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	set := make(map[uint]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		_, ok := set[v]
		if !ok {
			return false
		}
	}
	return true
}

func (sched *Scheduler) needReschedule(si *shardInfo) bool {
	if sched.sessionCtx != nil && sched.sessionCtx.si.isSame(si) {
		return false
	}
	if sched.sessionCtx != nil {
		log.Errorf("[Reschedule] old:%v. new:%v", sched.sessionCtx.si, si)
	} else {
		log.Errorf("[Reschedule] old:nil. new:%v", si)
	}
	return true
}

func (sched *Scheduler) reschedule(si *shardInfo) error {
	// 先停止旧bot session
	if err := sched.stopSessions(); err != nil {
		log.Errorf("Stop sessions failed. err:%v", err)
		return err
	}

	// 启动新bot session
	if err := sched.startSessions(si); err != nil {
		log.Errorf("Start sessions failed. err:%v", err)
		return err
	}
	return nil
}

// getAP 获取bot websocket gateway信息
func (sched *Scheduler) getAP() (*dto.WebsocketAP, error) {
	botToken := token.BotToken(sched.args.BotAppID, sched.args.BotToken)
	openAPI := botgo.NewOpenAPI(botToken).WithTimeout(3 * time.Second)
	ctx := context.Background()
	ap, err := openAPI.WS(ctx, nil, "")
	if err != nil {
		log.Errorf("Open api ws failed. err:%v", err)
		return nil, err
	}
	log.Infof("Get ap info:%+v", ap)
	return ap, nil
}

func (sched *Scheduler) calShard(allIns []base.Instance) (*shardInfo, error) {
	si := &shardInfo{}
	// 过滤有效实例，获取实例idx和数量
	selfIdx, validInsNum := sched.countValidIns(allIns)
	if selfIdx < 0 || validInsNum <= 0 {
		return si, nil
	}

	// 获取Bot Gateway的AP链接点信息
	var err error
	si.ap, err = sched.getAP()
	if err != nil {
		log.Errorf("Call getAP failed. err:%v", err)
		return nil, err
	}
	wsMinShardNum := uint(si.ap.Shards)
	// 取有效实例数和最小分区数中较大的值为分区数
	if validInsNum < wsMinShardNum {
		si.shardNum = wsMinShardNum
		// 计算当前实例需要处理的分区id列表
		round := wsMinShardNum / validInsNum
		for i := uint(0); i < round; i++ {
			// 将id一轮一轮的加入到列表中
			si.shardIDs = append(si.shardIDs, i*validInsNum+uint(selfIdx))
		}
		// 处理余数
		tmp := wsMinShardNum % validInsNum
		if tmp != 0 && tmp > uint(selfIdx) {
			si.shardIDs = append(si.shardIDs, round*validInsNum+uint(selfIdx))
		}
	} else {
		// 实例数即为分区数时，每个实例以自己的idx作为需要消费的分区id
		si.shardNum = validInsNum
		si.shardIDs = append(si.shardIDs, uint(selfIdx))
	}
	log.Infof("cal shard:%v", si)
	return si, nil
}

// countValidIns 计算有效实例，返回自己在有效实例中的idx和有效实例总数
func (sched *Scheduler) countValidIns(allIns []base.Instance) (int, uint) {
	var validInsNum uint
	selfIdx := -1
	for _, ins := range allIns {
		log.Debugf("[Instance] %v", ins.GetName())
		if !ins.IsValid() {
			// 跳过无效instance
			continue
		}
		if sched.isSelf(ins) {
			selfIdx = int(validInsNum)
		}
		validInsNum++
	}
	return selfIdx, validInsNum
}

func (sched *Scheduler) stopSessions() error {
	if sched.sessionCtx == nil {
		return nil
	}

	sched.sessionCtx.cancelFunc()
	sched.sessionCtx.wg.Wait()
	sched.sessionCtx = nil
	return nil
}

func (sched *Scheduler) startSessions(si *shardInfo) error {
	if sched.sessionCtx != nil {
		return nil
	}
	if !si.isValid() {
		log.Errorf("Invalid shard. Do not start session, shard:%+v", si)
		return nil
	}
	sessionCtx := &botSessionCtx{
		ctx: context.Background(),
		si:  si,
	}
	sessionCtx.ctx, sessionCtx.cancelFunc = context.WithCancel(sessionCtx.ctx)
	sched.sessionCtx = sessionCtx
	sched.sessionCtx.wg.Add(1)
	// 启动bot服务协程
	go func() {
		defer func() {
			sessionCtx.wg.Done()
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				buf = buf[:runtime.Stack(buf, false)]
				log.Errorf("[RunBot]err:%v, stack:\n%s", r, buf)
				// 如果故障，则退出进程（通常不可能进入到这里）
				os.Exit(-1)
			}
		}()
		err := sched.runBot(sessionCtx.ctx, si)
		if err != nil {
			if err != context.Canceled {
				panic(fmt.Sprintf("Run bot failed. shard:%+v, err:%v", si, err))
			}
		}
	}()
	return nil
}

func (sched *Scheduler) runBot(ctx context.Context, si *shardInfo) error {
	log.Infof("[BotStart] shard:%v", si)
	token := token.BotToken(sched.args.BotAppID, sched.args.BotToken)
	sm := NewSessionManager(ctx, token, &sched.args.Intent, si)
	if err := sm.Start(); err != nil {
		log.Errorf("session start failed. err:%v", err)
		return err
	}
	log.Infof("[BotStop] shard:%v", si)
	return nil
}

func (args *Args) isValid() bool {
	if args.Intent == 0 ||
		args.BotAppID == 0 ||
		args.BotToken == "" ||
		args.Cluster == nil {
		return false
	}
	return true
}
