package common

/**
常量池
*/

const (

	//目录
	JOB_SAVE_DIR   = "/cron/jobs/"   //任务保存目录
	JOB_KILLER_DIR = "/cron/killer/" //任务终止目录
	JOB_LOCK_DIR   = "/cron/lock/"   //任务分布式锁目录

	//任务事件
	JOB_EVENT_SAVE   = 1 //更新任务事件
	JOB_EVENT_DELETE = 2 //删除任务事件
)
