/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 10:56
 */
package main

import (
	"flag"
	"github.com/json-iterator/go"
	"gochat/api"
	"gochat/connect"
	"gochat/log"
	"gochat/site"
	"gochat/task"
	"runtime/pprof"

	//"flag"
	"fmt"
	//"gochat/api"
	//"gochat/connect"
	"gochat/logic"
	"gochat/proto"
	//"gochat/site"
	//"gochat/task"
	"os"
	"os/signal"
	"syscall"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func main2() {
	msg := proto.Msg{}
	//a := `{"ver":1,"op":3,"seq":"123","body":{"msg": "大家好", "roomId": 1}}`
	msg.Body = proto.RoomMsg{Msg: "大家好", RoomId: 1}
	msg.Ver = 1
	msg.Operation = 3
	msg.SeqId = "123"
	msgbyte, _ := json.Marshal(msg)
	fmt.Println(msg)
	fmt.Println(string(msgbyte))

	msg2 := proto.Msg{}
	body := &proto.RoomMsg{}
	msg2.Body = body
	err := json.Unmarshal(msgbyte, &msg2)
	if err != nil {
		log.Log.Errorf("logic,OnMessage fail,err:%s", err.Error())
		return
	}
	room, _ := msg2.Body.(*proto.RoomMsg)
	//bodyJ ,_:=json.Marshal(msg2.Body)
	//json.Unmarshal(bodyJ, &body)
	//msg2.Body = body
	fmt.Printf("%v\n", msg2)
	fmt.Print(111)
	fmt.Printf("%+v\n", *room)
	fmt.Printf("%#v\n", *room)
	log.Log.Infof("%#v", *room)

}

func main() {
	var module string
	flag.StringVar(&module, "module", "", "assign run module")
	flag.Parse()
	fmt.Println(fmt.Sprintf("start run %s module", module))

	f, err := os.Create(module + ".prof")
	if err != nil {
		log.Log.Fatal(err)
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		log.Log.Fatal(err)
	}
	defer pprof.StopCPUProfile()
	switch module {
	case "logic":
		logic.New().Run()
	case "connect":
		connect.New().Run()
	case "task":
		task.New().Run()
	case "api":
		api.New().Run()
	case "site":
		site.New().Run()
	default:
		fmt.Println("exiting,module param error!")
		return
	}
	fmt.Println(fmt.Sprintf("run %s module done!", module))
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGTSTP)
	<-quit
	fmt.Println("Server exiting")
}
