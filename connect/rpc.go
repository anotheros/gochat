/**
 * Created by lock
 * Date: 2019-08-12
 * Time: 23:36
 */
package connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/rcrowley/go-metrics"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/serverplugin"
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"gochat/tools"
	"strings"
	"sync"
	"time"
)

var logicRpcClient client.XClient
var once sync.Once

type RpcConnect struct {
}

var rpcConnectObj *RpcConnect

func (c *Connect) InitLogicRpcClient() (err error) {
	once.Do(func() {
		d := client.NewEtcdV3Discovery(
			config.Conf.Common.CommonEtcd.BasePath,
			config.Conf.Common.CommonEtcd.ServerPathLogic,
			[]string{config.Conf.Common.CommonEtcd.Host},
			nil,
		)
		logicRpcClient = client.NewXClient(config.Conf.Common.CommonEtcd.ServerPathLogic, client.Failtry, client.RandomSelect, d, client.DefaultOption)
		rpcConnectObj = new(RpcConnect)
	})
	if logicRpcClient == nil {
		return errors.New("get rpc client nil")
	}
	return
}

func (rpc *RpcConnect) Connect(connReq *proto.ConnectRequest) (uid int, err error) {
	reply := &proto.ConnectReply{}
	err = logicRpcClient.Call(context.Background(), "Connect", connReq, reply)
	if err != nil {
		log.Log.Fatalf("failed to call: %#v", err)
	}
	uid = reply.UserId
	log.Log.Infof("connect logic userId :%d", reply.UserId)
	return
}

func (rpc *RpcConnect) DisConnect(disConnReq *proto.DisConnectRequest) (err error) {
	reply := &proto.DisConnectReply{}
	if err = logicRpcClient.Call(context.Background(), "DisConnect", disConnReq, reply); err != nil {
		log.Log.Errorf("failed to call: %#v", err)
	}
	return
}

func (rpc *RpcConnect) CheckAuth(checkAuthReq *proto.CheckAuthRequest) (*proto.CheckAuthResponse, error) {
	reply := &proto.CheckAuthResponse{}
	err := logicRpcClient.Call(context.Background(), "CheckAuth", checkAuthReq, reply)
	if err != nil {
		log.Log.Fatalf("failed to call CheckAuth: %#v", err)
	}
	return reply, err
}

func (rpc *RpcConnect) OnMessage(mgRequest *proto.PushMsgRequest) (reply *proto.SuccessReply, err error) {

	err = logicRpcClient.Call(context.Background(), "OnMessage", mgRequest, &reply)
	if err != nil {
		log.Log.Errorf("=========failed to call OnMessage: %#v", err)
	}
	return
}

func (c *Connect) InitConnectRpcServer() (err error) {
	var network, addr string
	connectRpcAddress := strings.Split(config.Conf.Connect.ConnectRpcAddress.Address, ",")
	for _, bind := range connectRpcAddress {
		if network, addr, err = tools.ParseNetwork(bind); err != nil {
			log.Log.Panicf("InitConnectRpcServer ParseNetwork error : %s", err)
		}
		log.Log.Infof("Connect start run at-->%s:%s", network, addr)
		go c.createConnectRpcServer(network, addr)
	}
	return
}

var RedisClient *redis.Client


func (c *Connect) InitRedisClient() (err error) {
	redisOpt := tools.RedisOption{
		Address:  config.Conf.Common.CommonRedis.RedisAddress,
		Password: config.Conf.Common.CommonRedis.RedisPassword,
		Db:       config.Conf.Common.CommonRedis.Db,
	}
	RedisClient = tools.GetRedisInstance(redisOpt)
	if pong, err := RedisClient.Ping().Result(); err != nil {
		log.Log.Infof("RedisCli Ping Result pong: %s,  err: %s", pong, err)
	}
	return err
}

type RpcConnectPush struct {
}

func (rpc *RpcConnectPush) PushSingleMsg(ctx context.Context, pushMsgReq *proto.PushMsgRequest, successReply *proto.SuccessReply) (err error) {
	var (
		bucket  *Bucket
		channel *Channel
	)
	if pushMsgReq == nil {
		log.Log.Debugf("rpc PushSingleMsg() args:(%+v)", &pushMsgReq)
		return
	}
	bucket = DefaultServer.Bucket(pushMsgReq.UserId)
	channel = bucket.Channel(pushMsgReq.UserId)
	if channel == nil {
		log.Log.Warnf("DefaultServer Channel err nil ,args: %+v", pushMsgReq)
		return
	}
	err = channel.Send(pushMsgReq.Msg)
	if err != nil {
		log.Log.Errorf("channel.Push %#v", err)
	}

	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg
	//	log.Log.Infof("successReply:%+v", successReply)
	return
}

func (rpc *RpcConnectPush) PushRoomMsg(ctx context.Context, pushRoomMsgReq *proto.PushRoomMsgRequest, successReply *proto.SuccessReply) (err error) {
	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg
	log.Log.Infof("PushRoomMsg msg %+v", pushRoomMsgReq)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(pushRoomMsgReq)
	}
	return
}

func (rpc *RpcConnectPush) PushRoomCount(ctx context.Context, pushRoomMsgReq *proto.PushRoomMsgRequest, successReply *proto.SuccessReply) (err error) {
	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg

	log.Log.Infof("connect,PushRoomInfo msg %#v", &pushRoomMsgReq)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(pushRoomMsgReq)
	}
	return
}

func (rpc *RpcConnectPush) PushRoomInfo(ctx context.Context, pushRoomMsgReq *proto.PushRoomMsgRequest, successReply *proto.SuccessReply) (err error) {
	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg

	log.Log.Infof("connect,PushRoomInfo msg %+v", &pushRoomMsgReq)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(pushRoomMsgReq)
	}
	return
}

func (c *Connect) createConnectRpcServer(network string, addr string) {
	s := server.NewServer()
	addRegistryPlugin(s, network, addr)
	s.RegisterName(config.Conf.Common.CommonEtcd.ServerPathConnect, new(RpcConnectPush), fmt.Sprintf("%d", config.Conf.Common.CommonEtcd.ServerId))
	s.Serve(network, addr)
}

func addRegistryPlugin(s *server.Server, network string, addr string) {
	r := &serverplugin.EtcdV3RegisterPlugin{
		ServiceAddress: network + "@" + addr,
		EtcdServers:    []string{config.Conf.Common.CommonEtcd.Host},
		BasePath:       config.Conf.Common.CommonEtcd.BasePath,
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
	}
	err := r.Start()
	if err != nil {
		log.Log.Fatal(err)
	}
	s.Plugins.Add(r)
}
