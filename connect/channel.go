/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 15:18
 */
package connect

import (
	"github.com/gorilla/websocket"
)

//in fact, Channel it's a user Connect session
type Channel struct {
	Room      *Room
	Next      *Channel
	Prev      *Channel
	broadcast chan []byte
	userId    int
	conn      *websocket.Conn
}

func NewChannel(size int) (c *Channel) {
	c = new(Channel)
	c.broadcast = make(chan []byte, size)
	c.Next = nil
	c.Prev = nil
	return
}

func (ch *Channel) Push(msg []byte) (err error) {
	select {
	case ch.broadcast <- msg:
	default:
	}
	return
}
