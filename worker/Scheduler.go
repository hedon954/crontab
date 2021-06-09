package worker

import (
	"crontab/common"
	"fmt"
	"time"
)

/**
调度协程
*/

var (
	G_schedular *Scheduler
)

type Scheduler struct {
	jobEventChan      chan *common.JobEvent              //etcd 的调度事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan //任务调度表
	jobExecutingTable map[string]*common.JobExecuteInfo  //任务执行信息表
	jobResultChan     chan *common.JobExecuteResult      //任务执行结果队列
}

//启动调度协程
func (scheduler *Scheduler) scheduleLoop() {
	var (
		jobEvent         *common.JobEvent
		scheduleAfter    time.Duration
		scheduleTimer    *time.Timer
		jobExecuteResult *common.JobExecuteResult
	)

	//初始化一次(1s)
	scheduleAfter = scheduler.TrySchedule()

	//调度的延时定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//监听任务变化事件
	for {
		select {
		case jobEvent = <-scheduler.jobEventChan:
			//对内存维护的任务列表做增删改查
			scheduler.HandleJobEvent(jobEvent)
		case <-scheduleTimer.C: //最近的任务到期了
		case jobExecuteResult = <-scheduler.jobResultChan: //监听到任务执行结果
			scheduler.HandleJobResult(jobExecuteResult)

		}

		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		//重新调度间隔
		scheduleTimer.Reset(scheduleAfter)

	}

}

//处理任务事件
func (scheduler *Scheduler) HandleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		jobExisted      bool
		err             error
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		if jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE:
		if jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]; jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name)
		}
	}
}

//推送任务变化事件
func (scheduler *Scheduler) PushJobEven(jobEvent *common.JobEvent) {
	scheduler.jobEventChan <- jobEvent
}

//重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan  *common.JobSchedulePlan
		nowTime  time.Time
		nearTime *time.Time
	)

	nowTime = time.Now()

	//1. 遍历所有任务
	for _, jobPlan = range scheduler.jobPlanTable {
		//2. 过期的任务立即执行
		if !jobPlan.NextTime.After(nowTime) {
			scheduler.TryStartJob(jobPlan)
			//更新下次执行时间
			jobPlan.NextTime = jobPlan.Expr.Next(nowTime)
		}

		//3. 统计最近的要过期的任务的时间（N 秒后过期 == scheduleAfter）
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	//下次调度时间
	if nearTime != nil {
		scheduleAfter = (*nearTime).Sub(nowTime)
	} else {
		//没有任务，随便睡眠 1 s
		scheduleAfter = 1 * time.Second
	}

	return scheduleAfter
}

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	//调度和执行是 2 件事情
	//执行的任务可能运行很久，比如 1 个每秒调度 1 次的任务，1 次运行需要 1 分钟
	//这样1分钟会调度 60 次，但是只会执行 1 次
	//需要做去重，防止并发

	var (
		jobExecuteInfo *common.JobExecuteInfo
		jobExecuting   bool
	)

	//1. 如果任务正在执行，跳过本次任务
	if jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobPlan.Job.Name]; !jobExecuting {

		//2. 没有执行，就构建状态信息
		jobExecuteInfo = &common.JobExecuteInfo{
			Job:      jobPlan.Job,
			PlanTime: jobPlan.NextTime,
			RealTime: time.Now(),
		}

		//3. 保存状态信息
		scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo

		//4. 执行任务
		fmt.Println("开始执行任务：", jobExecuteInfo)
		G_executor.ExecuteJob(jobExecuteInfo)
	}
}

//处理任务执行结果
func (scheduler *Scheduler) HandleJobResult(result *common.JobExecuteResult) {
	//1. 删除执行状态
	delete(scheduler.jobExecutingTable, result.JobExecuteInfo.Job.Name)

	//2. 保存任务执行结果到 mongodb
	fmt.Println(result.JobExecuteInfo.Job, "任务执行结果：", string(result.Output))
}

//初始化调度器
func InitScheduler() (err error) {
	G_schedular = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1024),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult),
	}

	//启动
	go G_schedular.scheduleLoop()

	return
}
