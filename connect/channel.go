/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 15:18
 */
package connect

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"time"
)

//in fact, Channel it's a user Connect session
type Channel struct {
	Room   *Room
	Next   *Channel
	Prev   *Channel
	send   chan []byte
	userId int
	conn   *websocket.Conn
	hub    *Hub
	server *Server
}

func NewChannel(server *Server, hub *Hub, conn *websocket.Conn, userId int) (c *Channel) {
	c = new(Channel)
	c.send = make(chan []byte, server.Options.BroadcastSize)
	c.conn = conn
	c.server = server
	c.userId = userId
	c.hub = hub
	c.Next = nil
	c.Prev = nil
	return
}

func (ch *Channel) Push(msg []byte) (err error) {
	defer func() {

		if err := recover(); err != nil {
			log.Log.Errorf("push error : %#v",err)
			log.Log.Errorf("%#v" , ch)
		}

	}()
	select {
	case ch.send <- msg:
	default:
	}
	return
}

func (ch *Channel) onConnect() error {
	s := ch.server

	connReq := &proto.ConnectRequest{}
	connReq.UserId = ch.userId
	connReq.RoomId = 1 //TODO  这无需传入roomId ，链接层不需要知道room这个业务
	connReq.ServerId = config.Conf.Connect.ConnectBase.ServerId
	userId, err := s.operator.Connect(connReq)
	if err != nil {
		log.Log.Errorf("s.operator.Connect error %s", err.Error())
		return errors.New(fmt.Sprintf("s.operator.Connect error %s", err.Error()))
	}
	if userId == 0 {
		log.Log.Error("Invalid AuthToken ,userId empty")
		return errors.New("Invalid AuthToken ,userId empty")
	}
	log.Log.Infof("websocket rpc call return userId:%d,RoomId:%d", userId, connReq.RoomId)
	b := s.Bucket(userId)
	//insert into a bucket
	err = b.Put(userId, connReq.RoomId, ch)
	if err != nil {
		log.Log.Errorf("conn close err: %s", err.Error())
		//ch.conn.Close()
	}
	return nil
}
func (ch *Channel) writePump() {
	//PingPeriod default eq 54s
	s := ch.server
	ticker := time.NewTicker(s.Options.PingPeriod)
	defer func() {
		ticker.Stop()
		ch.conn.Close()
		close(ch.send)
		log.Log.Warnf("writePump defer")
	}()

	for {
		select {
		case message, ok := <-ch.send:
			//write data dead time , like http timeout , default 10s
			err := ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			if err != nil {
				log.Log.Warn(" ch.conn.SetWriteDeadline err :%s  ", err.Error())
				return
			}
			if !ok {
				log.Log.Warn("SetWriteDeadline not ok")
				ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := ch.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Log.Warn(" ch.conn.NextWriter err :%s  ", err.Error())
				return
			}

			log.Log.Infof("message write body:%s", string(message))
			n, err := w.Write(message)
			log.Log.Info(n)
			if err != nil {
				log.Log.Warn(" w.Write err :%s  ", err.Error())

			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			//heartbeat，if ping error will exit and close current websocket conn
			ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			log.Log.Infof("websocket.PingMessage :%v", websocket.PingMessage)
			if err := ch.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Log.Error(err.Error())
				return
			}
		}
	}
}

func (ch *Channel) readPump() {
	s := ch.server
	defer func() {
		log.Log.Info("=========start exec disConnect ...")
		if ch.Room == nil || ch.userId == 0 {
			log.Log.Info("==========roomId and userId eq 0")
			ch.conn.Close()
			ch.hub.Close()
			log.Log.Warnf("readPump defer")
			return
		}
		log.Log.Info("============exec disConnect ...")
		disConnectRequest := new(proto.DisConnectRequest)
		disConnectRequest.RoomId = ch.Room.Id
		disConnectRequest.UserId = ch.userId
		s.Bucket(ch.userId).DeleteChannel(ch)
		if err := s.operator.DisConnect(disConnectRequest); err != nil {
			log.Log.Warnf("DisConnect err :%s", err.Error())
		}
		ch.conn.Close()
		ch.hub.Close()
		log.Log.Warnf("readPump defer")
	}()
	log.Log.Info("readPump ...")
	ch.conn.SetReadLimit(s.Options.MaxMessageSize)
	ch.conn.SetReadDeadline(time.Now().Add(s.Options.WriteWait))
	ch.conn.SetPongHandler(func(string) error {
		log.Log.Warnf(">>>>>>>pong")
		ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
		return nil
	})

	for {
		messageTpye, message, err := ch.conn.ReadMessage()
		log.Log.Infof("=message in %s",string(message))
		log.Log.Infof("=message in %d",messageTpye)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Log.Errorf("===============readPump ReadMessage err:%s", err.Error())
				return
			}
			break
		}
		if message == nil {
			log.Log.Errorf("===============message == nil")
			continue
		}
		//log.Log.Info(messageType)

		//msg := &proto.Msg{}

		//msgString:=msg.Body
		//  消息发给 逻辑层
		//log.Log.Debug(string(message))
		pushMsgRequest := &proto.PushMsgRequest{Msg: message, UserId: ch.userId}
		//ch.hub.receive <- pushMsgRequest

		job := Job{Payload{msg: pushMsgRequest}}
		JobQueue <- job

	}
}
