package dbandmq

import (
	"fmt"
	"github.com/go-redis/redis"
	. "github.com/leyle/ginbase/consolelog"
)

type RedisOption struct {
	Host   string `json:"host" yaml:"host"`
	Port   string `json:"port" yaml:"port"`
	Passwd string `json:"passwd" yaml:"passwd"`
	DbNum  int    `json:"dbnum" yaml:"dbnum"`
}

func (o *RedisOption) Addr() string {
	return fmt.Sprintf("%s:%s", o.Host, o.Port)
}

func (o *RedisOption) String() string {
	return fmt.Sprintf("[%s:%s][%s],db[%d]", o.Host, o.Port, "******", o.DbNum)
}

func NewRedisClient(opt *RedisOption) (*redis.Client, error) {
	option := &redis.Options{
		Addr:     opt.Addr(),
		Password: opt.Passwd,
		DB:       opt.DbNum,
	}

	c := redis.NewClient(option)
	_, err := c.Ping().Result()
	if err != nil {
		Logger.Errorf("", "ping redis[%s]失败, %s", opt.String(), err.Error())
		return nil, err
	}

	Logger.Debugf("", "连接 redis[%s]成功", opt.String())
	return c, nil
}
