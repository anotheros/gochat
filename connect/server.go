/**
 * Created by lock
 * Date: 2019-08-10
 * Time: 18:32
 */
package connect

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"gochat/config"
	"gochat/proto"
	"gochat/tools"
	"time"
)

type Server struct {
	Buckets   []*Bucket
	Options   ServerOptions
	bucketIdx uint32
	operator  Operator
}

type ServerOptions struct {
	WriteWait       time.Duration
	PongWait        time.Duration
	PingPeriod      time.Duration
	MaxMessageSize  int64
	ReadBufferSize  int
	WriteBufferSize int
	BroadcastSize   int
}

func NewServer(b []*Bucket, o Operator, options ServerOptions) *Server {
	s := new(Server)
	s.Buckets = b
	s.Options = options
	s.bucketIdx = uint32(len(b))
	s.operator = o
	return s
}

//reduce lock competition, use google city hash insert to different bucket
func (s *Server) Bucket(userId int) *Bucket {
	userIdStr := fmt.Sprintf("%d", userId)
	idx := tools.CityHash32([]byte(userIdStr), uint32(len(userIdStr))) % s.bucketIdx
	return s.Buckets[idx]
}

func (s *Server) onConnect(auth string, ch *Channel) error {
	checkAuthRequest := &proto.CheckAuthRequest{
		AuthToken: auth,
	}

	reply, err := rpcConnectObj.CheckAuth(checkAuthRequest)
	if err != nil {
		logrus.Errorf("serverWs CheckAuth err:%s", err.Error())
		return errors.New("s.operator.Connect no authToken")
	}
	if reply.Code != config.SuccessReplyCode {
		logrus.Errorf("serverWs CheckAuth err:%s", reply.Code)
		return errors.New("s.operator.Connect no authToken")
	}

	connReq := &proto.ConnectRequest{}
	connReq.UserId = reply.UserId
	connReq.RoomId = 1 //TODO  这无需传入roomId ，链接层不需要知道room这个业务
	connReq.ServerId = config.Conf.Connect.ConnectBase.ServerId
	userId, err := s.operator.Connect(connReq)
	if err != nil {
		logrus.Errorf("s.operator.Connect error %s", err.Error())
		return errors.New(fmt.Sprintf("s.operator.Connect error %s", err.Error()))
	}
	if userId == 0 {
		logrus.Error("Invalid AuthToken ,userId empty")
		return errors.New("Invalid AuthToken ,userId empty")
	}
	logrus.Infof("websocket rpc call return userId:%d,RoomId:%d", userId, connReq.RoomId)
	b := s.Bucket(userId)
	//insert into a bucket
	err = b.Put(userId, connReq.RoomId, ch)
	if err != nil {
		logrus.Errorf("conn close err: %s", err.Error())
		ch.conn.Close()
	}
	return nil
}
func (s *Server) writePump(ch *Channel) {
	//PingPeriod default eq 54s
	ticker := time.NewTicker(s.Options.PingPeriod)
	defer func() {
		ticker.Stop()
		ch.conn.Close()
	}()

	for {
		select {
		case message, ok := <-ch.broadcast:
			//write data dead time , like http timeout , default 10s
			ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			if !ok {
				logrus.Warn("SetWriteDeadline not ok")
				ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := ch.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logrus.Warn(" ch.conn.NextWriter err :%s  ", err.Error())
				return
			}
			logrus.Infof("message write body:%s", message.Body)
			w.Write(message.Body)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			//heartbeat，if ping error will exit and close current websocket conn
			ch.conn.SetWriteDeadline(time.Now().Add(s.Options.WriteWait))
			logrus.Infof("websocket.PingMessage :%v", websocket.PingMessage)
			if err := ch.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Server) readPump(ch *Channel) {
	defer func() {
		logrus.Infof("start exec disConnect ...")
		if ch.Room == nil || ch.userId == 0 {
			logrus.Infof("roomId and userId eq 0")
			ch.conn.Close()
			return
		}
		logrus.Infof("exec disConnect ...")
		disConnectRequest := new(proto.DisConnectRequest)
		disConnectRequest.RoomId = ch.Room.Id
		disConnectRequest.UserId = ch.userId
		s.Bucket(ch.userId).DeleteChannel(ch)
		if err := s.operator.DisConnect(disConnectRequest); err != nil {
			logrus.Warnf("DisConnect err :%s", err.Error())
		}
		ch.conn.Close()
	}()
	logrus.Error("readPump ...")
	ch.conn.SetReadLimit(s.Options.MaxMessageSize)
	ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
	ch.conn.SetPongHandler(func(string) error {
		ch.conn.SetReadDeadline(time.Now().Add(s.Options.PongWait))
		return nil
	})

	for {
		messageType, message, err := ch.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("readPump ReadMessage err:%s", err.Error())
				return
			}
		}
		if message == nil {
			return
		}
		logrus.Info(messageType)


		//msg := &proto.Msg{}

		//msgString:=msg.Body
		// TODO 消息发给 逻辑层
		msgRequest := &proto.MsgRequest{Msg: message, UserId: ch.userId}
		reply, err := rpcConnectObj.OnMessage(msgRequest)
		logrus.Info(reply)
	}
}
