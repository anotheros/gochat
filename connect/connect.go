/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 18:18
 */
package connect

import (
	jsoniter "github.com/json-iterator/go"
	"gochat/config"
	"gochat/log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
var DefaultServer *Server

type Connect struct {
}

func New() *Connect {
	return new(Connect)
}

func (c *Connect) Run() {
	// get Connect layer config
	connectConfig := config.Conf.Connect

	//set the maximum number of CPUs that can be executing
	runtime.GOMAXPROCS(connectConfig.ConnectBucket.CpuNum)

	//init logic layer rpc client, call logic layer rpc server
	if err := c.InitLogicRpcClient(); err != nil {
		log.Log.Panicf("InitLogicRpcClient err:%s", err.Error())
	}
	//init Connect layer rpc server, logic client will call this
	Buckets := make([]*Bucket, connectConfig.ConnectBucket.CpuNum)
	for i := 0; i < connectConfig.ConnectBucket.CpuNum; i++ {
		Buckets[i] = NewBucket(BucketOptions{
			ChannelSize:   connectConfig.ConnectBucket.Channel,
			RoomSize:      connectConfig.ConnectBucket.Room,
			RoutineAmount: connectConfig.ConnectBucket.RoutineAmount,
			RoutineSize:   connectConfig.ConnectBucket.RoutineSize,
		})
	}

	operator := new(DefaultOperator)
	DefaultServer = NewServer(Buckets, operator, ServerOptions{
		WriteWait:       10 * time.Second,
		PongWait:        60 * time.Second,
		PingPeriod:      54 * time.Second,
		MaxMessageSize:  1024*16,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		BroadcastSize:   1024*16,
	})
	var dispatcher = NewDispatcher(config.MaxWorker)
	dispatcher.Run()
	go func (){http.ListenAndServe("localhost:3999", nil)}()
	//init Connect layer rpc server ,task layer will call this
	if err := c.InitConnectRpcServer(); err != nil {
		log.Log.Panicf("InitConnectRpcServer Fatal error: %s \n", err)
	}

	//start Connect layer server handler persistent connection
	if err := c.InitWebsocket(); err != nil {
		log.Log.Panicf("Connect layer InitWebsocket() error:  %s \n", err.Error())
	}

}
