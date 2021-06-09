package master

import (
	"crontab/common"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

//单例对象
var (
	G_apiServer *ApiServer
)

//任务的 HTTP 接口
type ApiServer struct {
	httpServer *http.Server
}

//初始化服务
func InitApiServer() error {
	var (
		mux *http.ServeMux
		listener net.Listener
		httpServer *http.Server
		staticDir http.Dir
		staticHandler http.Handler
		err error
	)

	//配置路由
	mux = http.NewServeMux()
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)

	//定义静态文件的存储目录
	staticDir = http.Dir(G_config.Webroot)
	staticHandler = http.FileServer(staticDir)

	//当请求 /index.html 的时候会被这个 mux 接收，然后交给 StripPrefix 过滤掉 /，剩下 index.html
	//然后 StripPrefix 会去 staticDir 下面找 index.html，也就是组成了 ./webroot/index.html
	mux.Handle("/", http.StripPrefix("/", staticHandler))


	//启动 HTTP 监听
	if listener, err = net.Listen("tcp", ":" + strconv.Itoa(G_config.ApiPort)); err != nil {
		fmt.Println("启动 HTTP 监听发生异常")
		return err
	}

	//创建一个 HTTP 服务
	httpServer = &http.Server{
		ReadTimeout: time.Duration(G_config.ApiReadTimeOut) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeOut) * time.Millisecond,
		Handler: mux,
	}

	//单例赋值
	G_apiServer = &ApiServer{
		httpServer: httpServer,
	}

	//启动 HTTP 服务端
	go httpServer.Serve(listener)

	return nil
}

/**
	接口 /job/kill ：强制终止某个任务

	请求格式：POST: jobName = job1
 */
func handleJobKill(writer http.ResponseWriter, request *http.Request) {
	var(
		bytes []byte
 		err error
		jobName string
	)
	//1. 解析表单
	if err = request.ParseForm(); err != nil {
		goto ERR
	}

	//2. 提取出表单中的 jobName
	jobName = request.PostForm.Get("jobName")

	//3. kill 掉 job
	if err = G_jobManager.KillJob(jobName); err != nil {
		goto ERR
	}

	//4. 正常结束
	if bytes, err = common.BuildResponse(0, "success", nil); err == nil {
		writer.Write(bytes)
	}
	return

ERR:
	//异常应答
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		writer.Write(bytes)
	}
}

/**
	接口 /job/list ：列出所有任务

	请求格式：GET
*/
func handleJobList(writer http.ResponseWriter, request *http.Request) {
	var (
		err error
		jobs []*common.Job
		bytes []byte
	)

	//获取任务列表
	if jobs, err = G_jobManager.ListJob(); err != nil {
		goto ERR
	}

	//正常应答
	if bytes,err = common.BuildResponse(0, "success", jobs); err == nil{
		writer.Write(bytes)
	}
	return

ERR:
	if bytes,err = common.BuildResponse(-1, "failed", nil); err == nil {
		writer.Write(bytes)
	}
}

/**
	接口 /job/save ：保存任务

	请求格式：POST job = {"name":"job1", "command": "echo hello", "cronExpr":"* * * * *"}
 */
func handleJobSave(writer http.ResponseWriter, request *http.Request) {
	var(
		err error
		postJob string
		job common.Job
		oldJob *common.Job
		bytes []byte
	)

	//1. 解析 POST 表单
	if err = request.ParseForm(); err != nil {
		goto ERR
	}

	//2. 获取表单的 job 字段
	postJob = request.PostForm.Get("job")

	//3. 反序列化 job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		goto ERR
	}

	//4. 将 job 存到 etcd
	if oldJob, err = G_jobManager.SaveJob(&job); err != nil {
		goto ERR
	}

	//5. 检查是否有旧值，有就返回
	if bytes,err = common.BuildResponse(0,"success", oldJob); err == nil {
		writer.Write(bytes)
	}

	//6. 正常退出
	return

	//7. 异常退出
	ERR:
		if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
			writer.Write(bytes)
		}
}

/**
	接口 /job/delete ：删除任务

	请求格式：POST /job/delete  jobName = job1
*/
func handleJobDelete(writer http.ResponseWriter, request *http.Request)  {
	var (
		err error
		jobName string
		oldJob *common.Job
		bytes []byte
	)

	//1. 解析 form 表单
	//POST: a=1&b=2&c=3
	if err = request.ParseForm(); err != nil {
		goto ERR
	}

	//2. 解析出 form 表单中的 jobName
	jobName = request.PostForm.Get("jobName")

	//3. 从 etcd 删除掉 job
	if oldJob, err = G_jobManager.DeleteJob(jobName); err != nil {
		goto ERR
	}

	//4. 正常应答
	if bytes,err = common.BuildResponse(0, "success", oldJob); err == nil {
		writer.Write(bytes)
	}

	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		writer.Write(bytes)
	}
}