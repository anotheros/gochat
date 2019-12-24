/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:50
 */
package task

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gochat/proto"
	"math/rand"
)

type PushParams struct {
	ServerId int
	UserId   int
	Msg      *proto.UserMsg
	RoomId   int
	SeqId    string
}

var pushChannel []chan *PushParams

func init() {
	pushChannel = make([]chan *PushParams, config.Conf.Task.TaskBase.PushChan)
}

func (task *Task) GoPush() {
	for i := 0; i < len(pushChannel); i++ {
		pushChannel[i] = make(chan *PushParams, config.Conf.Task.TaskBase.PushChanSize)
		go task.processSinglePush(pushChannel[i])
	}
}

func (task *Task) processSinglePush(ch chan *PushParams) {
	var arg *PushParams
	for {
		arg = <-ch
		task.pushSingleToConnect(arg.ServerId, arg.UserId, arg.Msg)
	}
}

func (task *Task) Push(msg string) {
	m := &proto.RedisMsg{}
	if err := json.Unmarshal([]byte(msg), m); err != nil {
		logrus.Infof(" json.Unmarshal err:%v ", err)
	}
	logrus.Infof("push msg info %s", m)
	switch m.Op {
	case config.OpSingleSend:
		pushChannel[rand.Int()%config.Conf.Task.TaskBase.PushChan] <- &PushParams{
			ServerId: m.ServerId,
			UserId:   m.UserId,
			Msg:      redisMsg2UserMsg(m),
		}
	case config.OpRoomSend:
		task.broadcastRoomToConnect(m.RoomId, m.SeqId, redisMsg2RoomMsg(m))
	case config.OpRoomCountSend:
		task.broadcastRoomCountToConnect(m.RoomId, m.Count)
	case config.OpRoomInfoSend:
		task.broadcastRoomInfoToConnect(m.RoomId, m.RoomUserInfo)
	}
}

func redisMsg2RoomMsg(*proto.RedisMsg) *proto.RoomMsg {

	return nil
}

func redisMsg2UserMsg(*proto.RedisMsg) *proto.UserMsg {

	return nil
}
