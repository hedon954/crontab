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
	jobEventChan chan *common.JobEvent			//etcd 的调度事件队列
	jobPlanTable map[string]*common.JobSchedulePlan
}

//启动调度协程
func (scheduler *Scheduler) scheduleLoop ()  {
	var(
		jobEvent *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
	)

	//初始化一次(1s)
	scheduleAfter = scheduler.TrySchedule()

	//调度的延时定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	//监听任务变化事件
	for{
		select {
		case jobEvent = <- scheduler.jobEventChan:
			//对内存维护的任务列表做增删改查
			scheduler.HandleJobEvent(jobEvent)
		case <- scheduleTimer.C:			//最近的任务到期了
		}

		//调度一次任务
		scheduleAfter = scheduler.TrySchedule()
		//重新调度间隔
		scheduleTimer.Reset(scheduleAfter)

	}



}

//处理任务事件
func (scheduler *Scheduler) HandleJobEvent (jobEvent *common.JobEvent) {
	var(
		jobSchedulePlan *common.JobSchedulePlan
		jobExisted bool
		err error
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		if jobSchedulePlan, err  = common.BuildJobSchedulePlan(jobEvent.Job); err != nil {
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
func (scheduler *Scheduler) PushJobEven(jobEvent *common.JobEvent)  {
	scheduler.jobEventChan <- jobEvent
}

//重新计算任务调度状态
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {
	var (
		jobPlan *common.JobSchedulePlan
		nowTime time.Time
		nearTime *time.Time
	)

	nowTime = time.Now()


	//1. 遍历所有任务
	for _,jobPlan = range scheduler.jobPlanTable{
		//2. 过期的任务立即执行
		if !jobPlan.NextTime.After(nowTime) {
			//TODO: 尝试执行任务
			fmt.Println("尝试执行任务：",jobPlan)
			//更新下次执行时间
			jobPlan.NextTime = jobPlan.Expr.Next(nowTime)
		}

		//3. 统计最近的要过期的任务的时间（N 秒后过期 == scheduleAfter）
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime){
			nearTime = &jobPlan.NextTime
		}
	}

	//下次调度时间
	if nearTime != nil {
		scheduleAfter = (*nearTime).Sub(nowTime)
	}else{
		//没有任务，随便睡眠 1 s
		scheduleAfter = 1 * time.Second
	}

	return scheduleAfter
}

//初始化调度器
func InitScheduler() (err error){
	G_schedular = &Scheduler{
		jobEventChan: make(chan *common.JobEvent, 1024),
		jobPlanTable: make(map[string]*common.JobSchedulePlan, 0),
	}

	//启动
	go G_schedular.scheduleLoop()

	return
}

