package test

import (
	"fmt"
	"testing"
	"time"
)

func TestChan(t *testing.T) {
	c := make(chan int,  3)

	go func() {
		c <- 48
		c <- 96
		time.Sleep(2 * time.Second)
		c <- 200
				time.Sleep(2 * time.Second)
		c <- 200
				time.Sleep(2 * time.Second)
		c <- 200
				time.Sleep(2 * time.Second)
		c <- 200
				time.Sleep(2 * time.Second)
		c <- 200

	}()

	time.Sleep(1 * time.Second)
	for v := range c {
		fmt.Println(v)
	}
	time.Sleep(100 * time.Second)
}
