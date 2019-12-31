/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 18:22
 */
package task

import (
	"gochat/config"
	"gochat/log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

type Task struct {
}

var task *Task
func New() *Task {
	task = new(Task)
	return task
}

func (task *Task) Run() {
	//read config
	taskConfig := config.Conf.Task
	runtime.GOMAXPROCS(taskConfig.TaskBase.CpuNum)
	//read from redis queue
	if err := task.InitSubscribeRedisClient(); err != nil {
		log.Log.Panicf("task init publishRedisClient fail,err:%s", err.Error())
	}
	//rpc call connect layer send msg
	if err := task.InitConnectRpcClient(); err != nil {
		log.Log.Panicf("task init InitConnectRpcClient fail,err:%s", err.Error())
	}
	//GoPush
	task.GoPush()
	go func (){http.ListenAndServe("localhost:5999", nil)}()
	var dispatcher = NewDispatcher(config.MaxWorker)
	dispatcher.Run()
}
