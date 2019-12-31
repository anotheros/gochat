/**
 * Created by lock
 * Date: 2019-08-12
 * Time: 19:23
 */
package proto

type RedisMsg struct {
	SeqId        string            `json:"seq"`
	Op           int               `json:"op"`
	ServerId     int               `json:"serverId,omitempty"`
	RoomId       int               `json:"roomId,omitempty"`
	FromUserId   int               `json:"fromUserId"`
	FromUserName string            `json:"fromUserName"`
	ToUserId     int               `json:"toUserId"`
	ToUserName   string            `json:"toUserName"`
	Msg          string            `json:"msg"`
	Count        int               `json:"count"`
	RoomUserInfo map[string]string `json:"roomUserInfo"`
	CreateTime   string            `json:"createTime"`
}

type TaskMessage struct {
	Channel string
	Pattern string
	Payload string
}

type RedisRoomInfo struct {
	Op           int               `json:"op"`
	RoomId       int               `json:"roomId,omitempty"`
	Count        int               `json:"count,omitempty"`
	RoomUserInfo map[string]string `json:"roomUserInfo"`
}

type RedisRoomCountMsg struct {
	Count int `json:"count,omitempty"`
	Op    int `json:"op"`
}

type SuccessReply struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
