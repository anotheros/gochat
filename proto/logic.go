/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:38
 */
package proto

type LoginRequest struct {
	Name     string
	Password string
}

type LoginResponse struct {
	Code      int
	AuthToken string
}

type GetUserInfoRequest struct {
	UserId int
}

type GetUserInfoResponse struct {
	Code     int
	UserId   int
	UserName string
}

type RegisterRequest struct {
	Name     string
	Password string
}

type RegisterReply struct {
	Code      int
	AuthToken string
}

type LogoutRequest struct {
	AuthToken string
}

type LogoutResponse struct {
	Code int
}

type CheckAuthRequest struct {
	AuthToken string
}

type CheckAuthResponse struct {
	Code     int
	UserId   int
	UserName string
}

type ConnectRequest struct {
	UserId   int `json:"userId"`
	RoomId   int `json:"roomId"`
	ServerId int `json:"serverId"`
}

type ConnectReply struct {
	UserId int
}

type DisConnectRequest struct {
	RoomId int
	UserId int
}

type DisConnectReply struct {
	Has bool
}

// 以下 返回给前端的格式，与接收前端的格式。是 Msg 的body
type UserMsg struct {
	FromUserId   int    `json:"fromUserId"`
	FromUserName string `json:"fromUserName"`
	ToUserId     int    `json:"toUserId"`
	ToUserName   string `json:"toUserName"`
	CreateTime   string `json:"createTime"`
	Msg          string `json:"msg"`
}
type RoomMsg struct {
	FromUserId   int    `json:"fromUserId"`
	FromUserName string `json:"fromUserName"`
	RoomId       int    `json:"roomId"`
	CreateTime   string `json:"createTime"`
	Msg          string `json:"msg"`
}

type RoomInfoMsg struct {
	RoomId       int               `json:"roomId,omitempty"`
	Count        int               `json:"count,omitempty"`
	RoomUserInfo map[string]string `json:"roomUserInfo"`
}

type Send struct {
	SeqId        string `json:"seq"`
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	FromUserId   int    `json:"fromUserId"`
	FromUserName string `json:"fromUserName"`
	ToUserId     int    `json:"toUserId"`
	ToUserName   string `json:"toUserName"`
	RoomId       int    `json:"roomId"`
	Op           int    `json:"op"`
	CreateTime   string `json:"createTime"`
}
