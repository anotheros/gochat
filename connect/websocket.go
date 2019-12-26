/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 15:19
 */
package connect

import (
	"github.com/gorilla/websocket"
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"net/http"
)

func (c *Connect) InitWebsocket() error {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c.serveWs(DefaultServer, w, r)
	})
	err := http.ListenAndServe(config.Conf.Connect.ConnectWebsocket.Bind, nil)
	return err
}

func (c *Connect) serveWs(server *Server, w http.ResponseWriter, r *http.Request) {

	vars := r.URL.Query()
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

	//default send size eq 512
	hub := newHub()
	go hub.run()
	userId := reply.UserId
	ch := NewChannel(server, hub, conn, userId)

	ch.hub.register <- ch

	//send data to websocket conn
	go ch.writePump()
	//get data from websocket conn
	go ch.readPump()
}
