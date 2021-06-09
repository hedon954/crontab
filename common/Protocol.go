package common

import (
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//定时任务
type Job struct {
	Name string `json:"name"`
	Command string	`json:"command"`
	CronExpr string `json:"cronExpr"`
}

//Job 事件
type JobEvent struct {
	EventType int  //SAVE, DELETE
	Job *Job
}

//任务调度计划
type JobSchedulePlan struct {
	Job *Job							//要调度的任务
	Expr *cronexpr.Expression			//解析好的 cron 表达式
	NextTime time.Time					//任务下次执行的时间
}

//接口的统一应答
type Response struct {
	Code int `json:"code"`
	Message string `json:"message"`
	Data interface{} `json:"data"`
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
	var(
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
func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey, JOB_SAVE_DIR)
}

//构建 Job 事件
func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job: job,
	}
}

//构建执行计划
func BuildJobSchedulePlan(job *Job) (*JobSchedulePlan, error) {
	var(
		jobSchedulePlan *JobSchedulePlan
		expr *cronexpr.Expression
		err error
	)

	//解析 cron 表达式
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return nil, err
	}

	//封装
	jobSchedulePlan = &JobSchedulePlan{
		Job: job,
		Expr: expr,
		NextTime: expr.Next(time.Now()),
	}

	return jobSchedulePlan, nil

}