/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:13
 */
package task

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/client"
	"gochat/config"
	"gochat/proto"
	"gochat/tools"
	"strconv"
	"strings"
)

var RpcConnectClientList map[int]client.XClient

func (task *Task) InitConnectRpcClient() (err error) {
	etcdConfig := config.Conf.Common.CommonEtcd
	d := client.NewEtcdV3Discovery(etcdConfig.BasePath, etcdConfig.ServerPathConnect, []string{etcdConfig.Host}, nil)
	if len(d.GetServices()) <= 0 {
		logrus.Panicf("no etcd server find!")
	}
	RpcConnectClientList = make(map[int]client.XClient, len(d.GetServices()))
	for _, connectConf := range d.GetServices() {
		connectConf.Value = strings.Replace(connectConf.Value, "=&tps=0", "", 1)
		serverId, error := strconv.ParseInt(connectConf.Value, 10, 8)
		if error != nil {
			logrus.Panicf("InitComets err，Can't find serverId. error: %s", error)
		}
		d := client.NewPeer2PeerDiscovery(connectConf.Key, "")
		RpcConnectClientList[int(serverId)] = client.NewXClient(etcdConfig.ServerPathConnect, client.Failtry, client.RandomSelect, d, client.DefaultOption)
		logrus.Infof("InitConnectRpcClient addr %s, v %+v", connectConf.Key, RpcConnectClientList[int(serverId)])
	}
	return
}

func (task *Task) pushSingleToConnect(serverId int, userId int, msg *proto.UserMsg) {
	//logrus.Infof("pushSingleToConnect Body %s", string(msg))
	pushMsgReq := &proto.PushMsgRequest{
		UserId: userId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpSingleSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	//todo lock
	err := RpcConnectClientList[serverId].Call(context.Background(), "PushSingleMsg", pushMsgReq, reply)
	if err != nil {
		logrus.Infof(" pushSingleToConnect Call err %v", err)
	}
	logrus.Infof("reply %s", reply.Msg)
}

func (task *Task) broadcastRoomToConnect(roomId int, seqId string, msg *proto.RoomMsg) {
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpRoomSend,
			SeqId:     seqId,
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	for _, rpc := range RpcConnectClientList {
		logrus.Infof("broadcastRoomToConnect rpc  %v", rpc)
		rpc.Call(context.Background(), "PushRoomMsg", pushRoomMsgReq, reply)
		logrus.Infof("reply %s", reply.Msg)
	}
}

func (task *Task) broadcastRoomCountToConnect(roomId, count int) {
	msg := &proto.RoomInfoMsg{
		Count: count,
	}
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpRoomCountSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	for _, rpc := range RpcConnectClientList {
		logrus.Infof("broadcastRoomCountToConnect rpc  %v", rpc)
		rpc.Call(context.Background(), "PushRoomCount", pushRoomMsgReq, reply)
		logrus.Infof("reply %s", reply.Msg)
	}
}

func (task *Task) broadcastRoomInfoToConnect(roomId int, roomUserInfo map[string]string) {
	msg := &proto.RoomInfoMsg{
		Count:        len(roomUserInfo),
		RoomUserInfo: roomUserInfo,
		RoomId:       roomId,
	}

	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpRoomInfoSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	for _, rpc := range RpcConnectClientList {
		logrus.Infof("broadcastRoomInfoToConnect rpc  %v", rpc)
		rpc.Call(context.Background(), "PushRoomInfo", pushRoomMsgReq, reply)
		logrus.Infof("broadcastRoomInfoToConnect rpc  reply %v", reply)
	}
}
