// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connect

import (
	"gochat/log"
	"gochat/proto"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.

	// Inbound messages from the clients.
	receive   chan *proto.PushMsgRequest
	closeChan chan struct{}
	// Register requests from the clients.
	register chan *User
	ns       map[int]*User
	// Unregister requests from clients.
	unregister chan *User
	server     *Server
}

func newHub() *Hub {
	return &Hub{
		//receive:    make(chan *proto.PushMsgRequest),
		server:     DefaultServer,
		register:   make(chan *User),
		unregister: make(chan *User),
		closeChan:  make(chan struct{}),
	}
}

func (h *Hub) run() {

	s := h.server
	ticker := time.NewTicker(s.Options.PingPeriod)
	for {
		select {
		case user := <-h.register:
			//client.onConnect()
			h.ns[user.userId] = user

		case user := <-h.unregister:

			defer func() {

				if err := recover(); err != nil {
					log.Log.Errorf("user := <-h.unregister : %#v", err)
				}

			}()
			if _, has := h.ns[user.userId]; has {
				delete(h.ns, user.userId)
			}
			user.conn.Close()
			close(user.out)
		case pushMsgRequest := <-h.receive:
			log.Log.Info(pushMsgRequest)
		//  收到消息
		/**reply, err := rpcConnectObj.OnMessage(pushMsgRequest)
		if err != nil {
			log.Log.Errorf("===========%#v", err)
		}
		log.Log.Info(reply)**/
		case <-ticker.C:
			go func() {
				for _, v := range h.ns {
					time.Sleep(10 * time.Millisecond)
					v.pool.Schedule(func() {
						v.writePing()
					})
				}
			}()

		case <-h.closeChan:
			return
		}
	}
}
func (h *Hub) Close() {
	h.closeChan <- struct{}{}
	close(h.closeChan)
	close(h.register)
	close(h.unregister)
	close(h.receive)
}
