// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connect

import (
	"gochat/log"
	"gochat/proto"
	"sync"
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
	register chan *Channel
	ns       map[int]*Channel
	// Unregister requests from clients.
	unregister chan *Channel
	server     *Server
	sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		//receive:    make(chan *proto.PushMsgRequest),
		server:     DefaultServer,
		ns:         make(map[int]*Channel),
		register:   make(chan *Channel),
		unregister: make(chan *Channel),
		closeChan:  make(chan struct{}),
	}
}

func (h *Hub) run() {

	//s := h.server
	//ticker := time.NewTicker(s.Options.PingPeriod)
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case client := <-h.register:
			client.onConnect()
			h.ns[client.userId] = client

		case client := <-h.unregister:

			defer func() {

				if err := recover(); err != nil {
					log.Log.Errorf("user := <-h.unregister : %#v", err)
				}

			}()
			if _, has := h.ns[client.userId]; has {
				delete(h.ns, client.userId)
			}
			client.conn.Close()
			close(client.out)
			client.onDisConnect()
		case pushMsgRequest := <-h.receive:
			log.Log.Info(pushMsgRequest)
		//  收到消息
		/**reply, err := rpcConnectObj.OnMessage(pushMsgRequest)
		if err != nil {
			log.Log.Errorf("===========%#v", err)
		}
		log.Log.Info(reply)**/
		case <-ticker.C:
			defer func() {

				if err := recover(); err != nil {
					log.Log.Errorf("push error : %#v", err)
				}

			}()
			//go func() {
			h.RLock()
			map1 := h.ns
			h.RUnlock()
				for _, v := range map1 {
					time.Sleep(10 * time.Millisecond)
					log.Log.Infof(nameConn(v.conn))
					v.pool.Schedule(func() {
						v.writePing()
					})
				}
			//}()

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
