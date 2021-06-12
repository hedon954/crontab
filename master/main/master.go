package main

import (
	"crontab/master"
	"flag"
	"runtime"
	"time"
)

var (
	confFile string //配置文件路径
)

//解析命令行参数
func initArgs() {
	//master -config ./master.json
	//master -h
	flag.StringVar(&confFile, "config", "./master.json", "指定 master.json")
	flag.Parse()
}

//初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {

	var (
		err error
	)

	//初始化命令行参数
	initArgs()

	//初始化线程
	initEnv()

	//加载配置文件
	if err = master.InitConfig(confFile); err != nil {
		panic(err)
	}

	//初始化 etcd 管理器
	if err = master.InitJobManager(); err != nil {
		panic(err)
	}

	//初始化日志管理器
	if err = master.InitLogManager(); err != nil {
		panic(err)
	}

	//启动 Api HTTP 服务
	if err = master.InitApiServer(); err != nil {
		panic(err)
	}

	//初始化服务发现
	if err = master.InitWorkerManager(); err != nil {
		panic(err)
	}

	for {
		time.Sleep(1 * time.Second)
	}

	return

}
