package connect

import (
	"fmt"
	"gochat/log"
	"gochat/proto"
)

var (
	// MaxQueue presents max queue counts
	MaxQueue = 1000
)
var JobQueue chan Job

func init() {

	JobQueue = make(chan Job, MaxQueue)

}

// Job represents the job to be run
type Job struct {
	Payload Payload
}

// Payload represents the post payload
type Payload struct {
	msg *proto.PushMsgRequest
}

// Worker represents the worker that executes the job
type Worker struct {
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
}

// NewWorker create a new worker
func NewWorker(workerPool chan chan Job) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
	}
}

// Start method starts the run loop for the worker, listening for a quit channel in
// case we need to stop it
func (w Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:

				//j, _ := json.Marshal(job)
				//log.Log.Info(string(j))
				pushMsgRequest := job.Payload.msg
				reply, err := rpcConnectObj.OnMessage(pushMsgRequest)
				if err != nil {
					log.Log.Errorf("===========%#v", err)
				}
				log.Log.Debug(reply)

			case <-w.quit:
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

// Dispatcher presents the job dispatcher
type Dispatcher struct {
	MaxWorkers int
	WorkerPool chan chan Job
}

// NewDispatcher create a new dispatcher
func NewDispatcher(maxWorkers int) *Dispatcher {
	return &Dispatcher{
		MaxWorkers: maxWorkers,
		WorkerPool: make(chan chan Job, maxWorkers),
	}
}

// Run will create some workers
func (d *Dispatcher) Run() {
	// starting n number of workers
	for i := 0; i < d.MaxWorkers; i++ {
		worker := NewWorker(d.WorkerPool)
		worker.Start()
	}

	go d.dispatch()
}

/*base on job to select a worker.*/
func (d *Dispatcher) dispatch0() {
	for {
		select {
		case job := <-JobQueue:
			go func(job Job) {
				jobChannel := <-d.WorkerPool
				jobChannel <- job
			}(job)
		}
	}
}

/*base on worker to select a job.*/
func (d *Dispatcher) dispatch() {
	fmt.Println("start dispatch.")
	for {
		// try to obtain a worker job channel that is available.
		// this will block until a worker is idle
		select {
		case jobChannel := <-d.WorkerPool:

			go func(jobChannel chan Job) {
				job := <-JobQueue
				// a job request has been received
				// dispatch the job to the worker job channel
				jobChannel <- job
			}(jobChannel)
		}
	}
}
