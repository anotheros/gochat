/**
 * Created by lock
 * Date: 2019-08-13
 * Time: 10:13
 */
package task

import (
	"github.com/go-redis/redis"
	"gochat/config"
	"gochat/log"
	"gochat/proto"
	"gochat/tools"
)

var RedisClient *redis.Client

func (task *Task) InitSubscribeRedisClient() (err error) {
	redisOpt := tools.RedisOption{
		Address:  config.Conf.Common.CommonRedis.RedisAddress,
		Password: config.Conf.Common.CommonRedis.RedisPassword,
		Db:       config.Conf.Common.CommonRedis.Db,
	}
	RedisClient = tools.GetRedisInstance(redisOpt)
	if pong, err := RedisClient.Ping().Result(); err != nil {
		log.Log.Infof("RedisClient Ping Result pong: %s,  err: %s", pong, err)
	}

	go func() {

		//redisSub := RedisClient.Subscribe(config.QueueName)
		redisSub := RedisClient.PSubscribe("*")
		ch := redisSub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					log.Log.Debugf("redisSub Channel !ok: %v", ok)
					break
				}
				//task.Push(msg.Payload)
				message := &proto.TaskMessage{Channel: msg.Channel, Pattern: msg.Pattern, Payload: msg.Payload}
				job := Job{Payload{msg: message}}
				JobQueue <- job
			}
		}
	}()
	return
}
