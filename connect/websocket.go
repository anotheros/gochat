/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 15:19
 */
package connect

import (
	"errors"
	"github.com/gobwas/ws"
	"github.com/mailru/easygo/netpoll"
	"gochat/config"
	"gochat/gopool"
	"gochat/log"
	"gochat/proto"
	"net"
	"time"
)

var (
	// Make pool of X size, Y sized work queue and one pre-spawned
	// goroutine.
	pool = gopool.NewPool(128, 100, 4)
	//chat = NewChat(pool)
	exit   = make(chan struct{})
	poller netpoll.Poller
)

func init() {
	var err error
	poller, err = netpoll.New(nil)
	if err != nil {
		log.Log.Fatal(err)
	}
}

func login(auth string) ( int, error) {
	checkAuthRequest := &proto.CheckAuthRequest{
		AuthToken: auth,
	}

	reply, err := rpcConnectObj.CheckAuth(checkAuthRequest)
	if err != nil {
		log.Log.Errorf("serverWs CheckAuth err:%s", err.Error())
		return 0,err
	}
	if reply.Code != config.SuccessReplyCode {
		log.Log.Errorf("serverWs CheckAuth err:%s", reply.Code)
		return 0,errors.New(string(reply.Code))
	}

	return reply.UserId,nil
}

// handle is a new incoming connection handler.
// It upgrades TCP connection to WebSocket, registers netpoll listener on
// it and stores it as a chat user in Chat instance.
//
// We will call it below within accept() loop.
func handle(conn net.Conn) {
	// NOTE: we wrap conn here to show that ws could work with any kind of
	// io.ReadWriter.
	safeConn := deadliner{conn, ioTimeout}
	var userId int
	u := ws.Upgrader{
		OnHeader: func(key, value []byte) error {
			log.Log.Infof("key %s,value %s", key, string(value))
			if string(key) != "Auth" {
				return nil
			}
			/**ok := httphead.ScanCookie(value, func(key, value []byte) bool {
				// Check session here or do some other stuff with cookies.
				// Maybe copy some values for future use.
				return false
			})
			if ok {
				return nil
			}**/
			if string(key) == "Auth" {

				auth := string(value)
				userIdd,err :=login(auth)
				if err != nil {
					return ws.RejectConnectionError(
						ws.RejectionReason("bad cookie"),
						ws.RejectionStatus(400),
					)
				}
				userId = userIdd
				return nil
			}
			return ws.RejectConnectionError(
				ws.RejectionReason("bad cookie"),
				ws.RejectionStatus(400),
			)
		},
	}
	// Zero-copy upgrade to WebSocket connection.
	hs, err := u.Upgrade(safeConn)
	if err != nil {
		log.Log.Infof("%s: upgrade error: %v", nameConn(conn), err)
		conn.Close()
		return
	}

	log.Log.Infof("%s: established websocket connection: %+v", nameConn(conn), hs)

	// Register incoming user in chat.
	//user := chat.Register(safeConn)

	user := NewChannel(DefaultServer, pool, userId, conn)
	// Create netpoll event descriptor for conn.
	// We want to handle only read events of it.
	desc := netpoll.Must(netpoll.HandleRead(conn))

	// Subscribe to events about conn.
	poller.Start(desc, func(ev netpoll.Event) {
		if ev&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
			// When ReadHup or Hup received, this mean that client has
			// closed at least write end of the connection or connections
			// itself. So we want to stop receive events about such conn
			// and remove it from the chat registry.
			poller.Stop(desc)
			//chat.Remove(user)
			return
		}
		// Here we can read some new message from connection.
		// We can not read it right here in callback, because then we will
		// block the poller's inner loop.
		// We do not want to spawn a new goroutine to read single message.
		// But we want to reuse previously spawned goroutine.
		pool.Schedule(func() {
			if err := user.Receive(); err != nil {
				// When receive failed, we can only disconnect broken
				// connection and stop to receive events about it.
				poller.Stop(desc)

			}
		})

	})
}

func nameConn(conn net.Conn) string {
	return conn.LocalAddr().String() + " > " + conn.RemoteAddr().String()
}
func (c *Connect) InitWebsocket() (err error) {
	// Create incoming connections listener.
	ln, err := net.Listen("tcp", config.Conf.Connect.ConnectWebsocket.Bind)
	if err != nil {
		log.Log.Fatal(err)
	}

	log.Log.Infof("websocket is listening on %s", ln.Addr().String())

	// Create netpoll descriptor for the listener.
	// We use OneShot here to manually resume events stream when we want to.
	acceptDesc := netpoll.Must(netpoll.HandleListener(
		ln, netpoll.EventRead|netpoll.EventOneShot,
	))

	// accept is a channel to signal about next incoming connection Accept()
	// results.
	accept := make(chan error, 1)

	// Subscribe to events about listener.
	poller.Start(acceptDesc, func(e netpoll.Event) {
		// We do not want to accept incoming connection when goroutine pool is
		// busy. So if there are no free goroutines during 1ms we want to
		// cooldown the server and do not receive connection for some short
		// time.
		err := pool.ScheduleTimeout(time.Millisecond, func() {
			conn, err := ln.Accept()
			if err != nil {
				accept <- err
				return
			}

			accept <- nil
			handle(conn)
		})
		if err == nil {
			err = <-accept
		}
		if err != nil {
			if err != gopool.ErrScheduleTimeout {
				goto cooldown
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				goto cooldown
			}

			log.Log.Fatalf("accept error: %v", err)

		cooldown:
			delay := 5 * time.Millisecond
			log.Log.Infof("accept error: %v; retrying in %s", err, delay)
			time.Sleep(delay)
		}

		poller.Resume(acceptDesc)
	})

	<-exit
	return
}

/**
func (c *Connect) serveWs(server *Server, w http.ResponseWriter, r *http.Request) {

	vars := r.URL.Query()
	var userId int

	userIdString := ""
	if vars["userId"] != nil {
		userIdString = vars["userId"][0]
	}

	if userIdString != "" {
		userIdd, err := strconv.Atoi(userIdString)
		if err != nil {
			log.Log.Error(userIdString)
			log.Log.Errorf("======%v", err)
			return
		}

		userId = userIdd
	} else {
		auth := vars["auth"][0]
		checkAuthRequest := &proto.CheckAuthRequest{
			AuthToken: auth,
		}

		reply, err := rpcConnectObj.CheckAuth(checkAuthRequest)
		if err != nil {
			log.Log.Errorf("serverWs CheckAuth err:%s", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}
		if reply.Code != config.SuccessReplyCode {
			log.Log.Errorf("serverWs CheckAuth err:%s", reply.Code)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))

			return
		}
		userId = reply.UserId
	}
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  server.Options.ReadBufferSize,
		WriteBufferSize: server.Options.WriteBufferSize,
	}
	//cross origin domain support
	upGrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upGrader.Upgrade(w, r, nil)

	if err != nil {
		log.Log.Errorf("serverWs err:%s", err.Error())
		return
	}

	hub := newHub()
	go hub.run()

	ch := NewChannel(server, hub, conn, userId)

	ch.hub.register <- ch

	//send data to websocket conn
	go ch.writePump()
	//get data from websocket conn
	go ch.readPump()
}
**/
