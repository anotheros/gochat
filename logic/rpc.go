/**
 * Created by lock
 * Date: 2019-08-12
 * Time: 15:52
 */
package logic

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gochat/config"
	"gochat/log"
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
		log.Log.Infof("register err:%s", err.Error())
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
		log.Log.Infof("register set redis token fail!")
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
		log.Log.Infof("register set redis token fail!")
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
	log.Log.Infof("logic CheckAuth ,authToken is:%s", authToken)
	if authToken == "" {
		return
	}
	sessionName := tools.GetSessionName(authToken)
	var userDataMap = map[string]string{}
	userDataMap, err = RedisSessClient.HGetAll(sessionName).Result()
	if err != nil {
		log.Log.Infof("check auth fail!,authToken is:%s", authToken)
		return err
	}
	if len(userDataMap) == 0 {
		log.Log.Infof("no this user session,authToken is:%s", authToken)
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
		log.Log.Infof("check auth fail!,authToken is:%s", authToken)
		return err
	}
	if len(userDataMap) == 0 {
		log.Log.Infof("no this user session,authToken is:%s", authToken)
		return
	}
	intUserId, _ := strconv.Atoi(userDataMap["userId"])
	sessIdMap := tools.GetSessionIdByUserId(intUserId)
	//del sess_map like sess_map_1
	err = RedisSessClient.Del(sessIdMap).Err()
	if err != nil {
		log.Log.Infof("logout del sess map error:%s", err.Error())
		return err
	}
	//del serverId
	logic := new(Logic)
	serverIdKey := logic.getUserKey(fmt.Sprintf("%d", intUserId))
	err = RedisSessClient.Del(serverIdKey).Err()
	if err != nil {
		log.Log.Infof("logout del server id error:%s", err.Error())
		return err
	}
	err = RedisSessClient.Del(sessionName).Err()
	if err != nil {
		log.Log.Infof("logout error:%s", err.Error())
		return err
	}
	reply.Code = config.SuccessReplyCode
	return
}

//TODO  自己处理消息
func (rpc *RpcLogic) OnMessage(ctx context.Context, msgreq *proto.PushMsgRequest, reply *proto.SuccessReply) (err error) {

	msg := proto.Msg{}
	err2 := json.Unmarshal(msgreq.Msg, &msg)
	fromUserId := msgreq.UserId

	if err2 != nil {
		log.Log.Warnf("logic,OnMessage :%#v", err2)
		return
	}

	reply.Code = config.FailReplyCode
	send := &proto.Send{}
	send.SeqId = msg.SeqId
	send.Op = msg.Operation

	switch msg.Operation {

	case config.OpRoomSend:
		bodyString, errr := json.Marshal(msg.Body)
		err = errr
		roomMsg := proto.RoomMsg{}
		err = json.Unmarshal(bodyString, &roomMsg)
		//if roomMsg, ok := msg.Body.(*proto.RoomMsg); ok {
		send.FromUserId = fromUserId
		send.Msg = roomMsg.Msg
		send.RoomId = roomMsg.RoomId
		send.CreateTime = roomMsg.CreateTime
		//}
	case config.OpSingleSend:
		bodyString, errr := json.Marshal(msg.Body)
		err = errr
		userMsg := proto.UserMsg{}
		err = json.Unmarshal(bodyString, &userMsg)
		//if userMsg, ok := msg.Body.(*proto.UserMsg); ok {
		send.FromUserId = fromUserId
		send.Msg = userMsg.Msg
		send.ToUserId = userMsg.ToUserId
		send.CreateTime = userMsg.CreateTime
		//}
	default:
		log.Log.Info("未匹配类型")
		return

	}

	u := new(dao.User)
	userName := u.GetUserNameByUserId(send.FromUserId)
	send.FromUserName = userName

	switch send.Op {

	case config.OpSingleSend:

		err = rpc.Push(ctx, send, reply)
	case config.OpRoomSend:

		err = rpc.PushRoom(ctx, send, reply)
	}
	return
}

/**
single send msg
*/
func (rpc *RpcLogic) Push(ctx context.Context, sendData *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode

	logic := new(Logic)
	userSidKey := logic.getUserKey(fmt.Sprintf("%d", sendData.ToUserId))
	if userSidKey == "" {
		return
	}
	serverId := RedisSessClient.Get(userSidKey).Val()
	if serverId == "" {
		return
	}
	var serverIdInt int
	serverIdInt, err = strconv.Atoi(serverId)
	if err != nil {
		log.Log.Errorf("logic,push parse int fail:%s", err.Error())
		return
	}
	log.Log.Infof("logic,push ,%d ,%#v ", sendData.ToUserId, sendData)

	redisMsg := &proto.RedisMsg{
		Op:           config.OpSingleSend,
		ServerId:     serverIdInt,
		ToUserId:     sendData.ToUserId,
		ToUserName:   sendData.ToUserName,
		FromUserId:   sendData.FromUserId,
		FromUserName: sendData.FromUserName,
		Msg:          sendData.Msg,
		SeqId:        sendData.SeqId,
		CreateTime:   sendData.CreateTime,
	}

	err = logic.RedisPublishChannel(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,redis publish err: %s", err.Error())
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
		log.Log.Errorf("logic,PushRoom redis hGetAll err:%s", err.Error())
		return
	}
	//if len(roomUserInfo) == 0 {
	//	return errors.New("no this user")
	//}

	sendData.RoomId = roomId
	sendData.Msg = args.Msg
	sendData.FromUserId = args.FromUserId
	sendData.FromUserName = args.FromUserName
	sendData.Op = config.OpRoomSend
	sendData.CreateTime = tools.GetNowDateTime()
	sendData.SeqId = args.SeqId

	redisMsg := &proto.RedisMsg{
		Op:           config.OpRoomSend,
		RoomId:       roomId,
		Count:        len(roomUserInfo),
		Msg:          args.Msg,
		RoomUserInfo: roomUserInfo,
		FromUserId:   sendData.FromUserId,
		FromUserName: sendData.FromUserName,
		SeqId:        sendData.SeqId,
		CreateTime:   sendData.CreateTime,
	}

	log.Log.Warnf("logic,pushRoom ,%d ,%#v ", roomId, redisMsg)

	err = logic.RedisPublishRoomInfo(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,PushRoom err:%s", err.Error())
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
		log.Log.Errorf("logic,Count err:%s", err.Error())
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
	var redisMsg = &proto.RedisMsg{
		Op:           config.OpRoomInfoSend,
		RoomId:       roomId,
		Count:        len(roomUserInfo),
		RoomUserInfo: roomUserInfo,
		SeqId:        tools.GetSnowflakeId(),
	}
	err = logic.RedisPushRoomInfo(redisMsg)
	if err != nil {
		log.Log.Errorf("logic,GetRoomInfo err:%s", err.Error())
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
		log.Log.Errorf("logic,connect args empty")
		return
	}
	logic := new(Logic)
	userId := args.UserId
	reply.UserId = userId
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(args.RoomId))

	userKey := logic.getUserKey(fmt.Sprintf("%d", reply.UserId))
	log.Log.Infof("logic redis set userKey:%s, serverId : %d", userKey, args.ServerId)
	validTime := config.RedisBaseValidTime * time.Second
	err = RedisClient.Set(userKey, args.ServerId, validTime).Err()
	if err != nil {
		log.Log.Warnf("logic set err:%s", err)
	}
	u := new(dao.User)
	userName := u.GetUserNameByUserId(userId)
	RedisClient.HSet(roomUserKey, fmt.Sprintf("%d", reply.UserId), userName)
	// add room user count ++
	RedisClient.Incr(logic.getRoomOnlineCountKey(fmt.Sprintf("%d", args.RoomId)))

	log.Log.Infof("logic rpc userId:%d", reply.UserId)
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
			log.Log.Warnf("HDel getRoomUserKey err : %s", err)
		}
		err = RedisClient.Del(fmt.Sprintf("%s%s", config.RedisPrefix, args.UserId)).Err()
		if err != nil {
			log.Log.Warnf("Del user err : %s", err)
		}
	}
	//below code can optimize send a signal to queue,another process get a signal from queue,then push event to websocket
	roomUserInfo, err := RedisClient.HGetAll(roomUserKey).Result()
	if err != nil {
		log.Log.Warnf("RedisCli HGetAll roomUserInfo key:%s, err: %s", roomUserKey, err)
	}

	var redisMsg = &proto.RedisMsg{
		Op:           config.OpRoomInfoSend,
		RoomId:       args.RoomId,
		Count:        len(roomUserInfo),
		RoomUserInfo: roomUserInfo,
		SeqId:        tools.GetSnowflakeId(),
	}
	if err = logic.RedisPushRoomInfo(redisMsg); err != nil {
		log.Log.Warnf("publish RedisPublishRoomCount err: %s", err.Error())
		return
	}
	return
}
