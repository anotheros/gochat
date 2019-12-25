/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:50
 */
package task

import (
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"math/rand"
)

type PushParams struct {
	ServerId int
	UserId   int
	Msg      *proto.UserMsg
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
		task.pushSingleToConnect(arg.ServerId, arg.SeqId, arg.UserId, arg.Msg)
	}
}

func (task *Task) Push(msg string) {
	m := &proto.RedisMsg{}
	if err := json.Unmarshal([]byte(msg), m); err != nil {
		log.Log.Infof(" json.Unmarshal err:%v ", err)
	}
	log.Log.Infof("push msg info %s", msg)
	switch m.Op {
	case config.OpSingleSend:
		pushChannel[rand.Int()%config.Conf.Task.TaskBase.PushChan] <- &PushParams{
			ServerId: m.ServerId,
			UserId:   m.ToUserId,
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

func redisMsg2RoomMsg(redisMsg *proto.RedisMsg) *proto.RoomMsg {

	roomMsg := &proto.RoomMsg{}
	roomMsg.Msg = redisMsg.Msg
	roomMsg.CreateTime = redisMsg.CreateTime
	roomMsg.RoomId = redisMsg.RoomId
	roomMsg.FromUserId = redisMsg.FromUserId
	roomMsg.FromUserName = redisMsg.FromUserName

	return roomMsg
}

func redisMsg2UserMsg(redisMsg *proto.RedisMsg) *proto.UserMsg {
	userMsg := &proto.UserMsg{}
	userMsg.Msg = redisMsg.Msg
	userMsg.CreateTime = redisMsg.CreateTime
	userMsg.ToUserId = redisMsg.ToUserId
	userMsg.ToUserName = redisMsg.ToUserName
	userMsg.FromUserId = redisMsg.FromUserId
	userMsg.FromUserName = redisMsg.FromUserName
	return userMsg
}
