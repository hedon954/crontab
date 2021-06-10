package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//定时任务
type Job struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	CronExpr string `json:"cronExpr"`
}

//Job 事件
type JobEvent struct {
	EventType int //SAVE, DELETE
	Job       *Job
}

//任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 //要调度的任务
	Expr     *cronexpr.Expression //解析好的 cron 表达式
	NextTime time.Time            //任务下次执行的时间
}

//任务执行信息表
type JobExecuteInfo struct {
	Job           *Job               //对应的任务
	PlanTime      time.Time          //理论上的调度时间
	RealTime      time.Time          //实际上的调度时间
	CancelContext context.Context    //任务 command 的上下文
	CancelFunc    context.CancelFunc //用于取消 command 执行的函数
}

//任务执行结果
type JobExecuteResult struct {
	JobExecuteInfo *JobExecuteInfo //任务执行状态
	Output         []byte          //任务执行输出
	Err            error           //任务执行错误信息
	StartTime      time.Time       //任务启动时间
	EndTime        time.Time       //任务完成时间
}

//任务执行日志
type JobLog struct {
	JobName      string `bson:"jobName"`      //任务名
	Command      string `bson:"command"`      //脚本命令
	Err          string `bson:"err"`          //异常信息
	Output       string `bson:"output"`       //任务输出
	PlanTime     int64  `bson:"planTime"`     //计划执行时间
	ScheduleTime int64  `bson:"scheduleTime"` //实际调度时间
	StartTime    int64  `bson:"startTime"`    //真实启动时间
	EndTime      int64  `bson:"endTime"`      //任务结束时间
}

//日志批次
type LogBatch struct {
	Logs []interface{} //多条日志
}

//接口的统一应答
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

//构建响应体
func BuildResponse(code int, message string, data interface{}) (bytes []byte, err error) {
	response := Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
	bytes, err = json.Marshal(response)
	return
}

//反序列化 Job
func UnmarshalJob(bytes []byte) (*Job, error) {
	var (
		err error
		job *Job
	)
	job = &Job{}
	if err = json.Unmarshal(bytes, job); err != nil {
		return nil, err
	}

	return job, nil
}

//从 etcd event key 中提取任务名称
func ExtractJobName(prefix string, jobKey string) string {
	return strings.TrimPrefix(jobKey, prefix)
}

//构建 Job 事件
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

//构建执行计划
func BuildJobSchedulePlan(job *Job) (*JobSchedulePlan, error) {
	var (
		jobSchedulePlan *JobSchedulePlan
		expr            *cronexpr.Expression
		err             error
	)

	//解析 cron 表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return nil, err
	}

	//封装
	jobSchedulePlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}

	return jobSchedulePlan, nil
}

//构建任务执行状态信息
func BuildJobExecuteInfo(plan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	cancelContext, cancelFunc := context.WithCancel(context.TODO())
	return &JobExecuteInfo{
		Job:           plan.Job,
		PlanTime:      plan.NextTime,
		RealTime:      time.Now(),
		CancelContext: cancelContext,
		CancelFunc:    cancelFunc,
	}
}
