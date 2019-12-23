/**
 * Created by lock
 * Date: 2019-08-12
 * Time: 15:52
 */
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gochat/logic/dao"
	"gochat/proto"
	"gochat/tools"
	"strconv"
	"time"
)

type RpcLogic struct {
}

func (rpc *RpcLogic) Register(ctx context.Context, args *proto.RegisterRequest, reply *proto.RegisterReply) (err error) {
	reply.Code = config.FailReplyCode
	u := new(dao.User)
	uData := u.CheckHaveUserName(args.Name)
	if uData.Id > 0 {
		return errors.New("this user name already have , please login !!!")
	}
	u.UserName = args.Name
	u.Password = args.Password
	userId, err := u.Add()
	if err != nil {
		logrus.Infof("register err:%s", err.Error())
		return err
	}
	if userId == 0 {
		return errors.New("register userId empty!")
	}
	//set token
	randToken := tools.GetRandomToken(32)
	sessionId := tools.CreateSessionId(randToken)
	userData := make(map[string]interface{})
	userData["userId"] = userId
	userData["userName"] = args.Name
	RedisSessClient.Do("MULTI")
	RedisSessClient.HMSet(sessionId, userData)
	RedisSessClient.Expire(sessionId, 86400*time.Second)
	err = RedisSessClient.Do("EXEC").Err()
	if err != nil {
		logrus.Infof("register set redis token fail!")
		return err
	}
	reply.Code = config.SuccessReplyCode
	reply.AuthToken = randToken
	return
}

func (rpc *RpcLogic) Login(ctx context.Context, args *proto.LoginRequest, reply *proto.LoginResponse) (err error) {
	reply.Code = config.FailReplyCode
	u := new(dao.User)
	userName := args.Name
	passWord := args.Password
	data := u.CheckHaveUserName(userName)
	if (data.Id == 0) || (passWord != data.Password) {
		return errors.New("no this user or password error!")
	}
	loginSessionId := tools.GetSessionIdByUserId(data.Id)
	//set token
	//err = redis.HMSet(auth, userData)
	randToken := tools.GetRandomToken(32)
	sessionId := tools.CreateSessionId(randToken)
	userData := make(map[string]interface{})
	userData["userId"] = data.Id
	userData["userName"] = data.UserName
	//check is login
	token, _ := RedisSessClient.Get(loginSessionId).Result()
	if token != "" {
		//logout already login user session
		oldSession := tools.CreateSessionId(token)
		err := RedisSessClient.Del(oldSession).Err()
		if err != nil {
			return errors.New("logout user fail!token is:" + token)
		}
	}
	RedisSessClient.Do("MULTI")
	RedisSessClient.HMSet(sessionId, userData)
	RedisSessClient.Expire(sessionId, 86400*time.Second)
	RedisSessClient.Set(loginSessionId, randToken, 86400*time.Second)
	err = RedisSessClient.Do("EXEC").Err()
	//err = RedisSessClient.Set(authToken, data.Id, 86400*time.Second).Err()
	if err != nil {
		logrus.Infof("register set redis token fail!")
		return err
	}
	reply.Code = config.SuccessReplyCode
	reply.AuthToken = randToken
	return
}

func (rpc *RpcLogic) GetUserInfoByUserId(ctx context.Context, args *proto.GetUserInfoRequest, reply *proto.GetUserInfoResponse) (err error) {
	reply.Code = config.FailReplyCode
	userId := args.UserId
	u := new(dao.User)
	userName := u.GetUserNameByUserId(userId)
	reply.UserId = userId
	reply.UserName = userName
	reply.Code = config.SuccessReplyCode
	return
}

func (rpc *RpcLogic) CheckAuth(ctx context.Context, args *proto.CheckAuthRequest, reply *proto.CheckAuthResponse) (err error) {
	reply.Code = config.FailReplyCode
	authToken := args.AuthToken
	logrus.Infof("logic CheckAuth ,authToken is:%s", authToken)
	if authToken == "" {
		return
	}
	sessionName := tools.GetSessionName(authToken)
	var userDataMap = map[string]string{}
	userDataMap, err = RedisSessClient.HGetAll(sessionName).Result()
	if err != nil {
		logrus.Infof("check auth fail!,authToken is:%s", authToken)
		return err
	}
	if len(userDataMap) == 0 {
		logrus.Infof("no this user session,authToken is:%s", authToken)
		return
	}
	intUserId, _ := strconv.Atoi(userDataMap["userId"])
	reply.UserId = intUserId
	userName, _ := userDataMap["userName"]
	reply.Code = config.SuccessReplyCode
	reply.UserName = userName
	return
}

func (rpc *RpcLogic) Logout(ctx context.Context, args *proto.LogoutRequest, reply *proto.LogoutResponse) (err error) {
	reply.Code = config.FailReplyCode
	authToken := args.AuthToken
	sessionName := tools.GetSessionName(authToken)

	var userDataMap = map[string]string{}
	userDataMap, err = RedisSessClient.HGetAll(sessionName).Result()
	if err != nil {
		logrus.Infof("check auth fail!,authToken is:%s", authToken)
		return err
	}
	if len(userDataMap) == 0 {
		logrus.Infof("no this user session,authToken is:%s", authToken)
		return
	}
	intUserId, _ := strconv.Atoi(userDataMap["userId"])
	sessIdMap := tools.GetSessionIdByUserId(intUserId)
	//del sess_map like sess_map_1
	err = RedisSessClient.Del(sessIdMap).Err()
	if err != nil {
		logrus.Infof("logout del sess map error:%s", err.Error())
		return err
	}
	//del serverId
	logic := new(Logic)
	serverIdKey := logic.getUserKey(fmt.Sprintf("%d", intUserId))
	err = RedisSessClient.Del(serverIdKey).Err()
	if err != nil {
		logrus.Infof("logout del server id error:%s", err.Error())
		return err
	}
	err = RedisSessClient.Del(sessionName).Err()
	if err != nil {
		logrus.Infof("logout error:%s", err.Error())
		return err
	}
	reply.Code = config.SuccessReplyCode
	return
}

//TODO  自己处理消息
func (rpc *RpcLogic) OnMessage(ctx context.Context, mgRequest *proto.MsgRequest, reply *proto.SuccessReply) (err error) {

	reply.Code = config.FailReplyCode
	logrus.Infof("logic,OnMessage %d : %s", mgRequest.UserId,string(mgRequest.Msg))
	var send proto.Send

	err =json.Unmarshal(mgRequest.Msg,&send)
	if err != nil {
		logrus.Errorf("logic,OnMessage fail,err:%s", err.Error())
		return
	}
	send.FromUserId = mgRequest.UserId

	u := new(dao.User)
	userName := u.GetUserNameByUserId(mgRequest.UserId)
	send.FromUserName = userName

	if send.RoomId !=0 {
		send.Op = config.OpRoomSend
	}
	if send.ToUserId != 0 {
		send.Op = config.OpSingleSend
	}
	j,_:=json.Marshal(send)
	switch  send.Op {

	case config.OpSingleSend:

		logrus.Warn("logic rpc single 209 "+string(j))
		err = rpc.Push(ctx,&send,reply)
	case config.OpRoomSend:
		logrus.Warn("212logic rpc room  "+string(j))
		err = rpc.PushRoom(ctx,&send,reply)
	}
	return
}

/**
single send msg
*/
func (rpc *RpcLogic) Push(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	sendData := args
	var bodyBytes []byte
	bodyBytes, err = json.Marshal(sendData)
	if err != nil {
		logrus.Errorf("logic,push msg fail,err:%s", err.Error())
		return
	}
	logic := new(Logic)
	userSidKey := logic.getUserKey(fmt.Sprintf("%d", sendData.ToUserId))
	if userSidKey == "" {
		return
	}
	serverId := RedisSessClient.Get(userSidKey).Val()
	var serverIdInt int
	serverIdInt, err = strconv.Atoi(serverId)
	if err != nil {
		logrus.Errorf("logic,push parse int fail:%s", err.Error())
		return
	}
	logrus.Infof("logic,push ,%d ,%s ", sendData.ToUserId, string(bodyBytes))
	err = logic.RedisPublishChannel(serverIdInt, sendData.ToUserId, bodyBytes)
	if err != nil {
		logrus.Errorf("logic,redis publish err: %s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return
}

/**
push msg to room
*/
func (rpc *RpcLogic) PushRoom(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	sendData := args
	roomId := sendData.RoomId
	logic := new(Logic)
	roomUserInfo := make(map[string]string)
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(roomId))
	roomUserInfo, err = RedisClient.HGetAll(roomUserKey).Result()
	if err != nil {
		logrus.Errorf("logic,PushRoom redis hGetAll err:%s", err.Error())
		return
	}
	//if len(roomUserInfo) == 0 {
	//	return errors.New("no this user")
	//}
	var bodyBytes []byte
	sendData.RoomId = roomId
	sendData.Msg = args.Msg
	sendData.FromUserId = args.FromUserId
	sendData.FromUserName = args.FromUserName
	sendData.Op = config.OpRoomSend
	sendData.CreateTime = tools.GetNowDateTime()
	bodyBytes, err = json.Marshal(sendData)
	if err != nil {
		logrus.Errorf("logic,PushRoom Marshal err:%s", err.Error())
		return
	}
	logrus.Warnf("logic,pushRoom ,%d ,%s ", roomId, string(bodyBytes))
	err = logic.RedisPublishRoomInfo(roomId, len(roomUserInfo), roomUserInfo, bodyBytes)
	if err != nil {
		logrus.Errorf("logic,PushRoom err:%s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return
}

/**
get room online person count
*/
func (rpc *RpcLogic) Count(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	roomId := args.RoomId
	logic := new(Logic)
	var count int
	count, err = RedisSessClient.Get(logic.getRoomOnlineCountKey(fmt.Sprintf("%d", roomId))).Int()
	err = logic.RedisPushRoomCount(roomId, count)
	if err != nil {
		logrus.Errorf("logic,Count err:%s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return
}

/**
get room info
*/
func (rpc *RpcLogic) GetRoomInfo(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	logic := new(Logic)
	roomId := args.RoomId
	roomUserInfo := make(map[string]string)
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(roomId))
	roomUserInfo, err = RedisClient.HGetAll(roomUserKey).Result()
	if len(roomUserInfo) == 0 {
		return errors.New("getRoomInfo no this user")
	}
	err = logic.RedisPushRoomInfo(roomId, len(roomUserInfo), roomUserInfo)
	if err != nil {
		logrus.Errorf("logic,GetRoomInfo err:%s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return
}

//TODO 业务处理，1.这里无需 处理登录 ，登录不是业务。
// 2.这里应该查询数据库 用户在哪个房间，与谁是好友。开始监听好友与房间的频道
// 3 这里是个demo
func (rpc *RpcLogic) Connect(ctx context.Context, args *proto.ConnectRequest, reply *proto.ConnectReply) (err error) {
	if args == nil {
		logrus.Errorf("logic,connect args empty")
		return
	}
	logic := new(Logic)
	userId := args.UserId
	reply.UserId = userId
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(args.RoomId))

	userKey := logic.getUserKey(fmt.Sprintf("%d", reply.UserId))
	logrus.Infof("logic redis set userKey:%s, serverId : %d", userKey, args.ServerId)
	validTime := config.RedisBaseValidTime * time.Second
	err = RedisClient.Set(userKey, args.ServerId, validTime).Err()
	if err != nil {
		logrus.Warnf("logic set err:%s", err)
	}
	u := new(dao.User)
	userName := u.GetUserNameByUserId(userId)
	RedisClient.HSet(roomUserKey, fmt.Sprintf("%d", reply.UserId), userName)
	// add room user count ++
	RedisClient.Incr(logic.getRoomOnlineCountKey(fmt.Sprintf("%d", args.RoomId)))

	logrus.Infof("logic rpc userId:%d", reply.UserId)
	return
}

func (rpc *RpcLogic) DisConnect(ctx context.Context, args *proto.DisConnectRequest, reply *proto.DisConnectReply) (err error) {
	logic := new(Logic)
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(args.RoomId))
	// room user count --
	if args.RoomId > 0 {
	     RedisClient.Decr(logic.getRoomOnlineCountKey(fmt.Sprintf("%d", args.RoomId))).Result()
	}
	// room login user--
	if args.UserId != 0 {
		err = RedisClient.HDel(roomUserKey, fmt.Sprintf("%d", args.UserId)).Err()
		if err != nil {
			logrus.Warnf("HDel getRoomUserKey err : %s", err)
		}
	}
	//below code can optimize send a signal to queue,another process get a signal from queue,then push event to websocket
	roomUserInfo, err := RedisClient.HGetAll(roomUserKey).Result()
	if err != nil {
		logrus.Warnf("RedisCli HGetAll roomUserInfo key:%s, err: %s", roomUserKey, err)
	}
	logrus.Warnf("logic,rpc redisMsg warn  ")
	if err = logic.RedisPushRoomInfo(args.RoomId, len(roomUserInfo), roomUserInfo); err != nil {
		logrus.Warnf("publish RedisPublishRoomCount err: %s", err.Error())
		return
	}
	return
}
