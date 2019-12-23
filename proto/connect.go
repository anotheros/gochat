/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:00
 */
package proto

/**
与前端的接口
 */
type Msg struct {
	Ver       int    `json:"ver"`  // protocol version
	Operation int    `json:"op"`   // operation for request
	SeqId     string `json:"seq"`  // sequence number chosen by client
	Body      []byte `json:"body"` // binary body bytes
}

/**
与逻辑层的
 */
type PushMsgRequest struct {
	UserId int
	Msg    Msg
}

type PushRoomMsgRequest struct {
	RoomId int
	Msg    Msg
}

type PushRoomCountRequest struct {
	RoomId int
	Count  int
}
