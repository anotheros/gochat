/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:13
 */
package task

import (
	"context"

	jsoniter "github.com/json-iterator/go"
	"github.com/smallnest/rpcx/client"
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"strconv"
	"strings"
)

var RpcConnectClientList map[int]client.XClient
var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (task *Task) InitConnectRpcClient() (err error) {
	etcdConfig := config.Conf.Common.CommonEtcd
	d := client.NewEtcdV3Discovery(etcdConfig.BasePath, etcdConfig.ServerPathConnect, []string{etcdConfig.Host}, nil)
	if len(d.GetServices()) <= 0 {
		log.Log.Panicf("no etcd server find!")
	}
	RpcConnectClientList = make(map[int]client.XClient, len(d.GetServices()))
	for _, connectConf := range d.GetServices() {
		connectConf.Value = strings.Replace(connectConf.Value, "=&tps=0", "", 1)
		serverId, error := strconv.ParseInt(connectConf.Value, 10, 8)
		if error != nil {
			log.Log.Panicf("InitComets err，Can't find serverId. error: %s", error)
		}
		d := client.NewPeer2PeerDiscovery(connectConf.Key, "")
		RpcConnectClientList[int(serverId)] = client.NewXClient(etcdConfig.ServerPathConnect, client.Failtry, client.RandomSelect, d, client.DefaultOption)
		log.Log.Debugf("InitConnectRpcClient addr %s, v %+v", connectConf.Key, RpcConnectClientList[int(serverId)])
	}
	return
}

func (task *Task) pushSingleToConnect(serverId int, seqId string, userId int, msg *proto.UserMsg) {
	//log.Log.Infof("pushSingleToConnect Body %s", string(msg))
	protoMsg := proto.Msg{
		Ver:       config.MsgVersion,
		Operation: config.OpSingleSend,
		SeqId:     seqId,
		Body:      msg,
	}
	byteMsg, err := json.Marshal(protoMsg)
	if err != nil {
		log.Log.Errorf(" pushSingleToConnect json err %#v", err)
	}
	pushMsgReq := &proto.PushMsgRequest{
		UserId: userId,
		Msg:    byteMsg,
	}
	reply := &proto.SuccessReply{}
	//todo lock
	err = RpcConnectClientList[serverId].Call(context.Background(), "PushSingleMsg", pushMsgReq, reply)
	if err != nil {
		log.Log.Errorf(" pushSingleToConnect Call err %#v", err)
	}
}

func (task *Task) broadcastRoomToConnect(roomId int, seqId string, msg *proto.RoomMsg) {
	protoMsg := proto.Msg{
		Ver:       config.MsgVersion,
		Operation: config.OpRoomSend,
		SeqId:     seqId,
		Body:      msg,
	}
	byteMsg, err := json.Marshal(protoMsg)
	if err != nil {
		log.Log.Errorf(" broadcastRoomToConnect json err %v", err)
	}
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg:    byteMsg,
	}
	reply := &proto.SuccessReply{}
	for _, rpc := range RpcConnectClientList {
		log.Log.Infof("broadcastRoomToConnect rpc  %#v", rpc)
		rpc.Call(context.Background(), "PushRoomMsg", pushRoomMsgReq, reply)
		log.Log.Infof("reply %s", reply.Msg)
	}
}

func (task *Task) broadcastRoomCountToConnect(roomId int,seqId string, count int) {
	msg := &proto.RoomInfoMsg{
		Count: count,
	}
	protoMsg := proto.Msg{
		Ver:       config.MsgVersion,
		Operation: config.OpRoomCountSend,
		SeqId:     seqId,
		Body:      msg,
	}
	byteMsg, err := json.Marshal(protoMsg)
	if err != nil {
		log.Log.Errorf(" broadcastRoomCountToConnect json err %#v", err)
	}
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg:    byteMsg,
	}
	reply := &proto.SuccessReply{}
	for _, rpc := range RpcConnectClientList {
		log.Log.Infof("broadcastRoomCountToConnect rpc  %#v", rpc)
		rpc.Call(context.Background(), "PushRoomCount", pushRoomMsgReq, reply)
		log.Log.Infof("reply %s", reply.Msg)
	}
}

func (task *Task) broadcastRoomInfoToConnect(roomId int,seqId string, roomUserInfo map[string]string) {
	msg := &proto.RoomInfoMsg{
		Count:        len(roomUserInfo),
		RoomUserInfo: roomUserInfo,
		RoomId:       roomId,
	}
	protoMsg := proto.Msg{
		Ver:       config.MsgVersion,
		Operation: config.OpRoomInfoSend,
		SeqId:     seqId,
		Body:      msg,
	}
	byteMsg, err := json.Marshal(protoMsg)
	if err != nil {
		log.Log.Errorf(" broadcastRoomInfoToConnect json err %#v", err)
	}
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg:    byteMsg,
	}
	reply := &proto.SuccessReply{}
	for _, rpc := range RpcConnectClientList {
		log.Log.Infof("broadcastRoomInfoToConnect rpc  %#v", rpc)
		rpc.Call(context.Background(), "PushRoomInfo", pushRoomMsgReq, reply)
		log.Log.Infof("broadcastRoomInfoToConnect rpc  reply %#v", reply)
	}
}
