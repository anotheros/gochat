// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package connect

import (
	"gochat/log"
	"gochat/proto"
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

	// Unregister requests from clients.
	unregister chan *Channel
}

func newHub() *Hub {
	return &Hub{
		receive:    make(chan *proto.PushMsgRequest),
		register:   make(chan *Channel),
		unregister: make(chan *Channel),
		closeChan:  make(chan struct{}),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			client.onConnect()
			log.Log.Info(client)
		case  <-h.unregister:

			//close(client.send)
			h.Close()
		case pushMsgRequest := <-h.receive:
			log.Log.Info(pushMsgRequest)
		//  收到消息
		/**reply, err := rpcConnectObj.OnMessage(pushMsgRequest)
		if err != nil {
			log.Log.Errorf("===========%#v", err)
		}
		log.Log.Info(reply)**/
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
