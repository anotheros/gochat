package connect

/**

import (
	"bytes"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"gochat/gopool"
	"gochat/log"
	"gochat/proto"
	"io"
	"sync"
)

// User represents user connection.
// It contains logic of receiving and sending messages.
// That is, there are no active reader or writer. Some other layer of the
// application should call Receive() to read user's incoming message.
type User struct {
	io     sync.Mutex
	conn   io.ReadWriteCloser
	pool   *gopool.Pool
	userId int
	name   string
	out    chan []byte
	//ticker *time.Ticker
	server *Server
}

func NewUser(server *Server, pool *gopool.Pool, userId int, conn io.ReadWriteCloser) (u *User) {
	u = new(User)
	u.server = server
	u.pool = pool
	u.userId = userId
	u.conn = conn
	//u.ticker = time.NewTicker(server.Options.PingPeriod)
	u.out = make(chan []byte, server.Options.BroadcastSize)
	return
}

// Receive reads next message from user's underlying connection.
// It blocks until full message received.
func (u *User) Receive() error {
	message, code, err := u.readRaw()
	if err != nil {
		u.conn.Close()
		return err
	}
	if code == ws.OpPong {

		return nil
	}
	if message == nil {
		// Handled some control message.
		return nil
	}
	//TODO
	pushMsgRequest := &proto.PushMsgRequest{Msg: message, UserId: u.userId}
	reply, err := rpcConnectObj.OnMessage(pushMsgRequest)
	if err != nil {
		log.Log.Errorf("===========%#v", err)
	}
	log.Log.Debug(reply)
	return nil
}

func (u *User) Send(p []byte) (err error) {
	u.io.Lock()
	defer u.io.Unlock()
	 u.pool.Schedule(u.writer)

	u.out <- p
	return
}

// readRequests reads json-rpc request from connection.
// It takes io mutex.
func (u *User) readRaw() ([]byte, ws.OpCode, error) {
	u.io.Lock()
	defer u.io.Unlock()

	h, code, err := wsutil.ReadClientData(u.conn)
	if err != nil {
		return nil, code, err
	}
	if code.IsControl() {
		return nil, code, nil //TODO
	}

	return h, code, nil
}

func (u *User) write(x interface{}) error {
	w := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	u.io.Lock()
	defer u.io.Unlock()

	if err := encoder.Encode(x); err != nil {
		return err
	}

	return w.Flush()
}

func (u *User) writePing()  {
	w := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpPing)

	u.io.Lock()
	defer u.io.Unlock()
	err := w.Flush()
	if err != nil {
		hubping.unregister <- u
	}
	return
}

func (u *User) writeRaw(p []byte) error {
	u.io.Lock()
	defer u.io.Unlock()

	_, err := u.conn.Write(p)

	return err
}

func (u *User) writer() {

	w := wsutil.NewWriter(u.conn, ws.StateServerSide, ws.OpText)
	u.io.Lock()
	defer u.io.Unlock()
	b := bytes.Buffer{}
	for bts := range u.out {
		b.Write(bts)
	}
	_,err := w.Write(b.Bytes())
	if err != nil {
		hubping.unregister <- u
	}

	return
}
**/
