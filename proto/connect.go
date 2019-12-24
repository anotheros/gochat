/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:00
 */
package proto



/**
逻辑层 推 connect
*/
type PushMsgRequest struct {
	UserId int
	Msg    []byte
}

type PushRoomMsgRequest struct {
	RoomId int
	Msg    []byte
}

type PushRoomCountRequest struct {
	RoomId int
	Count  int
}
