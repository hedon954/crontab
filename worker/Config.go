package worker

import (
	"encoding/json"
	"io/ioutil"
)

/**
初始化配置文件 worker.json
*/

var (
	G_config *Config
)

type Config struct {
	EtcdEndpoints         []string `json:"etcdEndPoints"`
	EtcdDiaTimeOut        int      `json:"etcdDiaTimeOut"`
	MongodbURI            string   `json:"mongodbURI"`
	MongodbConnectTimeOut int      `json:"mongodbConnectTimeOut"`
	JogLogBatchSize       int      `json:"jogLogBatchSize"`
	JobLogCommitTimeOut   int      `json:"jobLogCommitTimeOut"`
}

func InitConfig(filename string) error {

	var (
		contents []byte
		err      error
		config   Config
	)

	//读取配置文件内容
	if contents, err = ioutil.ReadFile(filename); err != nil {
		return err
	}

	//反序列化
	config = Config{}
	if err = json.Unmarshal(contents, &config); err != nil {
		return err
	}

	//单例赋值
	G_config = &config

	return nil

}
