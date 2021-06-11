package master

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

/**
	初始化 master.json 配置文件
 */

//单例
var(
	G_config *Config
)

type Config struct {
	ApiPort 				int   		`json:"apiPort"`
	ApiReadTimeOut 			int			`json:"apiReadTimeOut"`
	ApiWriteTimeOut 		int 		`json:"apiWriteTimeOut"`
	EtcdEndPoints 			[]string 	`json:"etcdEndPoints"`
	EtcdDialTimeOut 		int 		`json:"etcdDialTimeOut"`
	MongodbURI            	string   	`json:"mongodbURI"`
	MongodbConnectTimeOut 	int      	`json:"mongodbConnectTimeOut"`
	Webroot 				string 		`json:"webroot"`
}

//初始化配置
func InitConfig(filename string) error {
	var(
		err error
		content []byte
		config Config
	)

	//1. 读取配置文件
	if content, err = ioutil.ReadFile(filename); err != nil {
		return err
	}

	//2. JSON 反序列化
	if err = json.Unmarshal(content, &config); err != nil {
		return err
	}

	//3. 单例赋值
	G_config = &config

	fmt.Println(G_config)

	return nil
}