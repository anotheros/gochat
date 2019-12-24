/**
 * Created by lock
 * Date: 2019-08-09
 * Time: 10:56
 */
package main

import (
	"encoding/json"
	"flag"
	"gochat/api"
	"gochat/connect"
	"gochat/site"
	"gochat/task"

	//"flag"
	"fmt"
	"github.com/sirupsen/logrus"
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

func main2() {
	var send proto.Send
	a := `{"op":3,"msg":"123","roomId":1}`
	err := json.Unmarshal([]byte(a), &send)
	if err != nil {
		logrus.Errorf("logic,OnMessage fail,err:%s", err.Error())
		return
	}
	fmt.Print(send)

	push(&send)
}

func push(send *proto.Send) {
	sendData := send
	var bodyBytes2 []byte
	bodyBytes2, err := json.Marshal(sendData)
	if err != nil {
		logrus.Errorf("logic,push msg fail,err:%s", err.Error())
		return
	}
	logrus.Info(string(bodyBytes2))
}

func main() {
	var module string
	flag.StringVar(&module, "module", "", "assign run module")
	flag.Parse()
	fmt.Println(fmt.Sprintf("start run %s module", module))
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
