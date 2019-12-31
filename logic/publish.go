/**
 * Created by lock
 * Date: 2019-08-12
 * Time: 15:44
 */
package logic

import (
	"bytes"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/rcrowley/go-metrics"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/serverplugin"
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"gochat/tools"
	"strings"
	"time"
)

var RedisClient *redis.Client
var RedisSessClient *redis.Client

func (logic *Logic) InitPublishRedisClient() (err error) {
	redisOpt := tools.RedisOption{
		Address:  config.Conf.Common.CommonRedis.RedisAddress,
		Password: config.Conf.Common.CommonRedis.RedisPassword,
		Db:       config.Conf.Common.CommonRedis.Db,
	}
	RedisClient = tools.GetRedisInstance(redisOpt)
	if pong, err := RedisClient.Ping().Result(); err != nil {
		log.Log.Infof("RedisCli Ping Result pong: %s,  err: %s", pong, err)
	}
	//this can change use another redis save session data
	RedisSessClient = RedisClient
	return err
}

func (logic *Logic) InitRpcServer() (err error) {
	var network, addr string
	// a host multi port case
	rpcAddressList := strings.Split(config.Conf.Logic.LogicBase.RpcAddress, ",")
	for _, bind := range rpcAddressList {
		if network, addr, err = tools.ParseNetwork(bind); err != nil {
			log.Log.Panicf("InitLogicRpc ParseNetwork error : %s", err.Error())
		}
		log.Log.Infof("logic start run at-->%s:%s", network, addr)
		go logic.createRpcServer(network, addr)
	}
	return
}

func (logic *Logic) createRpcServer(network string, addr string) {
	s := server.NewServer()
	logic.addRegistryPlugin(s, network, addr)
	// serverId must be unique
	err := s.RegisterName(config.Conf.Common.CommonEtcd.ServerPathLogic, new(RpcLogic), fmt.Sprintf("%d", config.Conf.Common.CommonEtcd.ServerId))
	if err != nil {
		log.Log.Errorf("register error:%s", err.Error())
	}
	s.RegisterOnShutdown(func(s *server.Server) {
		s.UnregisterAll()
	})
	s.Serve(network, addr)
}

func (logic *Logic) addRegistryPlugin(s *server.Server, network string, addr string) {
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

func (logic *Logic) RedisPublishChannel(redisMsg *proto.RedisMsg) (err error) {
	redisMsgStr, err := json.Marshal(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,RedisPublishChannel Marshal err:%s", err.Error())
		return err
	}
	redisChannel := fmt.Sprintf(config.UserQueueName +"%d", redisMsg.ToUserId)
	if err := RedisClient.Publish(redisChannel, redisMsgStr).Err(); err != nil {
		log.Log.Errorf("logic,RedisPublishChannel err:%s", err.Error())
		return err
	}
	return
}

func (logic *Logic) RedisPublishRoomInfo(redisMsg *proto.RedisMsg) (err error) {

	redisMsgByte, err := json.Marshal(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,RedisPublishRoomInfo redisMsg error : %s", err.Error())
		return
	}
	redisChannel := fmt.Sprintf(config.RoomQueueName +"%d", redisMsg.ToUserId)
	err = RedisClient.Publish(redisChannel, redisMsgByte).Err()
	if err != nil {
		log.Log.Errorf("logic,RedisPublishRoomInfo redisMsg error : %s", err.Error())
		return
	}
	return
}

func (logic *Logic) RedisPushRoomCount(roomId int, count int) (err error) {
	var redisMsg = &proto.RedisMsg{
		Op:     config.OpRoomCountSend,
		RoomId: roomId,
		Count:  count,
		SeqId:tools.GetSnowflakeId(),
	}
	redisMsgByte, err := json.Marshal(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,RedisPushRoomCount redisMsg error : %s", err.Error())
		return
	}
	redisChannel := fmt.Sprintf(config.RoomQueueName +"%d", redisMsg.ToUserId)
	err = RedisClient.Publish(redisChannel, redisMsgByte).Err()
	if err != nil {
		log.Log.Errorf("logic,RedisPushRoomCount redisMsg error : %s", err.Error())
		return
	}
	return
}

func (logic *Logic) RedisPushRoomInfo(redisMsg *proto.RedisMsg) (err error) {
	redisMsgByte, err := json.Marshal(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,RedisPushRoomInfo redisMsg error : %s", err.Error())
		return
	}
	redisChannel := fmt.Sprintf(config.RoomQueueName +"%d", redisMsg.ToUserId)
	err = RedisClient.Publish(redisChannel, redisMsgByte).Err()
	if err != nil {
		log.Log.Errorf("logic,RedisPushRoomInfo redisMsg error : %s", err.Error())
		return
	}
	return
}

func (logic *Logic) getRoomUserKey(authKey string) string {
	var returnKey bytes.Buffer
	returnKey.WriteString(config.RedisRoomPrefix)
	returnKey.WriteString(authKey)
	return returnKey.String()
}

func (logic *Logic) getRoomOnlineCountKey(authKey string) string {
	var returnKey bytes.Buffer
	returnKey.WriteString(config.RedisRoomOnlinePrefix)
	returnKey.WriteString(authKey)
	return returnKey.String()
}

func (logic *Logic) getUserKey(authKey string) string {
	var returnKey bytes.Buffer
	returnKey.WriteString(config.RedisPrefix)
	returnKey.WriteString(authKey)
	return returnKey.String()
}
