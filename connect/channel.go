/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 15:18
 */
package connect

import (
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"gochat/config"
	"gochat/gopool"
	"gochat/log"
	"gochat/proto"
	"net"
	"sync"
)

//in fact, Channel it's a user Connect session
type Channel struct {
	Room *Room
	Next *Channel
	Prev *Channel

	io         sync.RWMutex
	rpcLock         sync.Mutex
	writerOnce sync.Once
	conn       net.Conn
	pool       *gopool.Pool
	userId     int
	name       string
	out        chan []byte
	//ticker *time.Ticker
	server *Server
}

func NewChannel(server *Server, pool *gopool.Pool, userId int, conn net.Conn) (c *Channel) {
	c = new(Channel)
	c.server = server
	c.pool = pool
	c.userId = userId
	c.conn = conn
	//u.ticker = time.NewTicker(server.Options.PingPeriod)
	c.out = make(chan []byte, server.Options.BroadcastSize)
	c.server = server
	c.userId = userId
	c.Next = nil
	c.Prev = nil
	return
}

func (ch *Channel) onDisConnect() error {
	s := ch.server
	disConnReq := &proto.DisConnectRequest{}
	disConnReq.UserId = ch.userId
	err := s.operator.DisConnect(disConnReq)
	return err
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

// Receive reads next message from user's underlying connection.
// It blocks until full message received.
func (u *Channel) Receive() error {
	message, code, err := u.readRaw()
	if err != nil {
		u.conn.Close()
		return err
	}
	if code == ws.OpPong {
		log.Log.Info(">>>>>>>>pong----------")
		return nil
	}
	if code == ws.OpPing {
		log.Log.Info(">>>>>>>>ping----------")
		u.writePong()
		return nil
	}
	if message == nil {
		// Handled some control message.
		return nil
	}
	//TODO
	log.Log.Infof("===########===%s",string(message))
	u.rpcLock.Lock()
	defer  u.rpcLock.Unlock()
	pushMsgRequest := &proto.PushMsgRequest{Msg: message, UserId: u.userId}
	reply, err := rpcConnectObj.OnMessage(pushMsgRequest)
	if err != nil {
		log.Log.Errorf("===========%#v", err)
	}
	log.Log.Debugf("%#v",reply)


	return nil
}

func (u *Channel) Send(p []byte) (err error) {
	defer func() {

		if err := recover(); err != nil {
			log.Log.Errorf("push error : %#v", err)
			log.Log.Errorf("%#v", u)
		}

	}()
	//u.io.Lock()
	//defer u.io.Unlock()
	//u.writerOnce.Do(func() {
		u.pool.Schedule(func() {u.writeRaw(p)})
	//})

	//u.out <- p

	return
}

// readRequests reads json-rpc request from connection.
// It takes io mutex.
func (u *Channel) readRaw() ([]byte, ws.OpCode, error) {
	u.io.RLock()
	defer u.io.RUnlock()

	h, code, err := wsutil.ReadClientData(u.conn)
	if err != nil {
		return nil, code, err
	}
	if code.IsControl() {
		return nil, code, nil //TODO
	}

	return h, code, nil
}

func (u *Channel) write(x interface{}) error {
	w := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	u.io.RLock()
	defer u.io.RUnlock()

	if err := encoder.Encode(x); err != nil {
		return err
	}

	return w.Flush()
}

func (u *Channel) writePing() {


	log.Log.Info(nameConn(u.conn) + "ping>>>>>>>>>>>")

	u.io.RLock()
	defer u.io.RUnlock()
	err := wsutil.WriteServerMessage(u.conn, ws.OpPing, nil)
	if err != nil {
		log.Log.Error(err.Error())
		Hubping.unregister <- u
	}
	return
}

func (u *Channel) writePong() {


	log.Log.Info("pong>>>>>>>")
	u.io.RLock()
	defer u.io.RUnlock()
	err := wsutil.WriteServerMessage(u.conn, ws.OpPong, nil)
	if err != nil {
		log.Log.Error(err.Error())
		Hubping.unregister <- u
	}
	return
}

func (u *Channel) writeRaw(p []byte)  {
	u.io.RLock()
	defer u.io.RUnlock()
	buf := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	buf.Write(p)
	err := buf.Flush()
	if err != nil {
		log.Log.Error(err.Error())
		Hubping.unregister <- u
		return
	}
	return
}

func (u *Channel) writer() {

	buf := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	//u.io.Lock()
	//defer u.io.Unlock()
	for bts := range u.out {
		b := bts
		buf.Write(b)
		err := buf.Flush()
		if err != nil {
			log.Log.Error(err.Error())
			Hubping.unregister <- u
			return
		}
	}

	return
}
