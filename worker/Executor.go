package worker

import (
	"context"
	"crontab/common"
	"os/exec"
	"time"
)

/**
任务执行器
*/

var (
	G_executor *Executor
)

type Executor struct {
}

//执行任务
func (executor *Executor) ExecuteJob(jobExecuteInfo *common.JobExecuteInfo) {
	//启动一个协程去执行任务
	go func() {

		var (
			command        *exec.Cmd
			combinedOutput []byte
			result         *common.JobExecuteResult
			jobLock        *JobLock
			err            error
		)

		result = &common.JobExecuteResult{
			JobExecuteInfo: jobExecuteInfo,
			Output:         make([]byte, 0),
		}

		//TODO: 争抢一个分布式锁
		jobLock = G_jobManager.CreateJobLock(jobExecuteInfo.Job.Name)
		err = jobLock.TryLock()

		//TODO: 释放锁
		defer jobLock.UnLock()

		if err != nil {
			result.Err = err
			result.EndTime = time.Now()
		} else {
			//记录任务开始时间
			result.StartTime = time.Now()

			//构建 shell 命令
			command = exec.CommandContext(context.TODO(), "/bin/bash", "-c", jobExecuteInfo.Job.Command)

			//执行 shell 命令
			combinedOutput, err = command.CombinedOutput()

			//记录任务结束时间
			result.EndTime = time.Now()
			result.Output = combinedOutput
			result.Err = err
		}

		//把任务执行结果返回给 Scheduler，Scheduler 会从 JobExecutingTable 中删除该任务
		executor.PushJobExecuteResult(result)
	}()
}

//回传任务执行结果
func (executor *Executor) PushJobExecuteResult(result *common.JobExecuteResult) {
	G_schedular.jobResultChan <- result
}

//初始化执行器
func InitExecutor() (err error) {
	G_executor = &Executor{}
	return nil
}
