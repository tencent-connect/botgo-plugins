// Package schedule bot集群调度器，根据北极星服务实例数量，以及频道AP信息中的
// 最小分区数，实时计算分区数量和分区号，并启动bot服务
package schedule

import (
	"context"
	"errors"
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

const (
	// DftWatchInterval 默认轮询间隔1分钟
	DftWatchInterval = time.Minute
	// MaxShardNum 最大分区数 10000
	MaxShardNum = uint32(10000)
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

	// 以下为可选参数

	// WatchInterval 调度轮询间隔，该间隔时间会触发一次调度检查，拉取AP信息计算是否需要调度，
	// 以支持单次调度失败时后续能够自动恢复，以及基础侧更新AP最小分区数的场景下能够自动按照最新
	// 的AP最小分区数进行重新分区
	WatchInterval time.Duration
	// MinShardNum 最小分区数，不能超过MaxShardNum，调度时取MinShardNum和AP信息中的Shards的较大值作为分区总数
	MinShardNum uint32
}

// Scheduler 调度器对象，通过NewScheduler构造对象，提供调度接口
type Scheduler struct {
	args          *Args
	localInstance base.Instance
	sessionCtx    *botSessionCtx
}

// shardInfo bot分区信息
type shardInfo struct {
	// shardIDs 需要处理的分区id列表
	shardIDs []uint32
	// shardNum 分区总数
	shardNum uint32
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

// NewArgs 获取参数
func NewArgs(cluster base.Cluster, botAppID uint64, botToken string, intent dto.Intent) *Args {
	return &Args{
		Cluster:  cluster,
		BotAppID: botAppID,
		BotToken: botToken,
		Intent:   intent,
	}
}

// New 创建调度器对象
func New(args *Args) (*Scheduler, error) {
	// 校验参数是否合法
	if !args.isValid() {
		return nil, fmt.Errorf("invalid args %+v", args)
	}
	localArgs := *args
	if localArgs.WatchInterval == 0 {
		// 采用默认参数
		localArgs.WatchInterval = DftWatchInterval
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ins, err := args.Cluster.GetLocalInstance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get local ins failed. err:%v", err)
	}

	return &Scheduler{
		args:          &localArgs,
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
	ticker := time.NewTicker(sched.args.WatchInterval)
	wc, err := sched.args.Cluster.Watch(context.Background())
	if err != nil {
		return err
	}
	for {
		if sched.IsExitSchedule() {
			break
		}
		select {
		case wr, ok := <-wc:
			if !ok || wr.Err != nil {
				time.Sleep(time.Second)
				// TODO 这里!ok场景是对端channel关闭，可以考虑退出进程重启
				continue
			}
			if err := sched.sharding(); err != nil {
				time.Sleep(time.Second)
				continue
			}
		case <-ticker.C:
			// 定时器到期，主动做一次sharding，里面会查询最新AP信息决定是否需要进行重新分区调度
			if err := sched.sharding(); err != nil {
				time.Sleep(time.Second)
				continue
			}
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
	return sched.localInstance.GetID() == ins.GetID()
}

func (s *shardInfo) isValid() bool {
	return len(s.shardIDs) > 0 && s.shardNum > 0
}

func (s *shardInfo) isSame(si *shardInfo) bool {
	return shardsEqual(s.shardIDs, si.shardIDs) && s.shardNum == si.shardNum
}

func shardsEqual(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	set := make(map[uint32]bool, len(a))
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

func (sched *Scheduler) getMinShardNum(ap *dto.WebsocketAP, validInsNum uint32) (uint32, error) {
	if ap.Shards == 0 {
		return 0, errors.New("invalid ap shards")
	}
	if ap.Shards < sched.args.MinShardNum {
		return sched.args.MinShardNum, nil
	}
	return ap.Shards, nil
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
	minShardNum, err := sched.getMinShardNum(si.ap, validInsNum)
	if err != nil {
		log.Errorf("getMinShardNum failed. err:%v", err)
		return nil, err
	}
	si.shardNum = minShardNum
	// 计算当前实例需要处理的分区id列表
	round := minShardNum / validInsNum
	for i := uint32(0); i < round; i++ {
		// 将id一轮一轮的加入到列表中
		si.shardIDs = append(si.shardIDs, i*validInsNum+uint32(selfIdx))
	}
	// 处理余数
	tmp := minShardNum % validInsNum
	if tmp != 0 && tmp > uint32(selfIdx) {
		si.shardIDs = append(si.shardIDs, round*validInsNum+uint32(selfIdx))
	}
	log.Infof("cal shard:%v", si)
	return si, nil
}

// countValidIns 计算有效实例，返回自己在有效实例中的idx和有效实例总数
func (sched *Scheduler) countValidIns(allIns []base.Instance) (int, uint32) {
	var validInsNum uint32
	selfIdx := -1
	for _, ins := range allIns {
		log.Debugf("[Instance] %v", ins.GetID())
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
		args.Cluster == nil ||
		args.MinShardNum > MaxShardNum {
		return false
	}
	return true
}
