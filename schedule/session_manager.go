// Package schedule 本文件内主要实现session链接管理功能
package schedule

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/errs"
	"github.com/tencent-connect/botgo/token"
	"github.com/tencent-connect/botgo/websocket"
)

// canNotResumeErrSet 不能进行 resume 操作的错误码
var canNotResumeErrSet = map[int]bool{
	errs.CodeConnCloseErr:   true,
	errs.CodeInvalidSession: true,
}

// sessionHolder 持有并维护session和对应websocket
type sessionHolder struct {
	session dto.Session
	ws      websocket.WebSocket
	mgr     *SessionManager
	stopped bool
}

// SessionManager session manager 实现，支持指定si
type SessionManager struct {
	ctx     context.Context
	si      *shardInfo
	token   *token.Token
	holders []*sessionHolder
	intents *dto.Intent
	// holderChan 用于接收待建立ws链接的session
	holderChan chan *sessionHolder
}

// NewSessionManager 新建SessionManager，如果要关闭session，可以Cancel该ctx
func NewSessionManager(ctx context.Context, token *token.Token,
	intents *dto.Intent, si *shardInfo) *SessionManager {
	return &SessionManager{
		ctx:     ctx,
		token:   token,
		intents: intents,
		si:      si,
	}
}

// Start 会按照传入的si分区信息启动对应session链接
func (mgr *SessionManager) Start() error {
	startInterval := calcInterval(mgr.si.ap.SessionStartLimit.MaxConcurrency)
	fmt.Printf("[ws/session] will start %d/%d sessions and per session start interval is %s\n",
		len(mgr.si.shardIDs), mgr.si.shardNum, startInterval)
	// 按照shards数量初始化，用于启动连接的管理
	mgr.holderChan = make(chan *sessionHolder, len(mgr.si.shardIDs))
	for _, sid := range mgr.si.shardIDs {
		holder := mgr.newHolder(sid)
		mgr.holders = append(mgr.holders, holder)
		mgr.holderChan <- holder
	}
	// 监听ctx以及holderChannel
	for {
		select {
		case h := <-mgr.holderChan:
			time.Sleep(time.Millisecond * 100)
			h.start()
			time.Sleep(startInterval)
		case <-mgr.ctx.Done():
			// ctx cancel，关闭所有session链接
			for _, h := range mgr.holders {
				h.stop()
			}
			return nil
		}
	}
}

func (mgr *SessionManager) newHolder(sid uint) *sessionHolder {
	return &sessionHolder{
		session: dto.Session{
			URL:     mgr.si.ap.URL,
			Token:   *mgr.token,
			Intent:  *mgr.intents,
			LastSeq: 0,
			Shards: dto.ShardConfig{
				ShardID:    uint32(sid),
				ShardCount: uint32(mgr.si.shardNum),
			},
		},
		mgr: mgr,
	}
}

func (holder *sessionHolder) stop() {
	holder.stopped = true
	if holder.ws != nil {
		holder.ws.Close()
	}
}

func (holder *sessionHolder) start() {
	if holder.stopped {
		return
	}
	holder.ws = websocket.ClientImpl.New(holder.session)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				buf = buf[:runtime.Stack(buf, false)]
				fmt.Printf("[SessionServe]err:%v, stack:\n%s\n", r, buf)
				// 如果故障，则退出进程（通常不可能进入到这里）
				os.Exit(-1)
			}
		}()
		holder.serve()
	}()
}

func (holder *sessionHolder) connectAndListen() {
	if err := holder.ws.Connect(); err != nil {
		fmt.Printf("[ws/session][%v] Connect err %+v\n", holder.session.Shards.ShardID, err)
		return
	}
	var err error
	// 如果 session id 不为空，则执行的是 resume 操作，如果为空，则执行的是 identify 操作
	if holder.session.ID != "" {
		err = holder.ws.Resume()
	} else {
		// 初次鉴权
		err = holder.ws.Identify()
	}
	if err != nil {
		fmt.Printf("[ws/session][%v] Identify/Resume err %+v\n", holder.session.Shards.ShardID, err)
		return
	}
	fmt.Printf("[ws/session][%v] connected\n", holder.session.Shards.ShardID)
	if err := holder.ws.Listening(); err != nil {
		fmt.Printf("[ws/session][%v] Listening err %+v\n", holder.session.Shards.ShardID, err)
		// 对于不能够进行重连的session，需要清空 session id 与 seq
		if canNotResume(err) {
			currentSession := holder.ws.Session()
			currentSession.ID = ""
			currentSession.LastSeq = 0
		}
		return
	}
}

func (holder *sessionHolder) serve() {
	holder.connectAndListen()
	if !holder.stopped {
		fmt.Printf("[ws/session][%v] reconnecting\n", holder.session.Shards.ShardID)
		// 稍微sleep 100ms再尝试重连
		time.Sleep(time.Millisecond * 100)
		// 将 session 放到 session chan 中，用于启动新的连接，当前连接退出
		holder.mgr.holderChan <- holder
	} else {
		fmt.Printf("[ws/session][%v] exiting\n", holder.session.Shards.ShardID)
	}
}

// canNotResume 是否是不能够 resume 的错误
func canNotResume(err error) bool {
	e := errs.Error(err)
	if flag, ok := canNotResumeErrSet[e.Code()]; ok {
		return flag
	}
	return false
}

// calcInterval 根据并发要求，计算连接启动间隔
func calcInterval(maxConcurrency uint32) time.Duration {
	// maxConcurrency 代表的是每 5s 可以连多少个请求
	f := math.Round(5 / float64(maxConcurrency))
	return time.Duration(f) * time.Second
}
