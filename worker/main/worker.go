package main

import (
	"crontab/worker"
	"flag"
	"runtime"
	"time"
)


var(
	confFile string //配置文件路径
)

//解析命令行参数
func initArgs()  {
	//worker -config ./worker.json
	//worker -h
	flag.StringVar(&confFile, "config", "./worker.json", "指定 worker.json")
	flag.Parse()
}

//初始化线程数量
func initEnv()  {
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

	//初始化配置文件
	if err = worker.InitConfig(confFile); err != nil {
		panic(err)
	}

	//启动调度器
	if err = worker.InitScheduler(); err != nil {
		panic(err)
	}

	//初始化 etcd 管理器
	if err  = worker.InitJobManager(); err != nil {
		panic(err)
	}

	//启动任务监听
	if err = worker.G_jobManager.WatchJobs(); err != nil {
		panic(err)
	}

	for  {
		time.Sleep(1 * time.Second)
	}

}
