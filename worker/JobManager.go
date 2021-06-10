package worker

import (
	"context"
	"crontab/common"
	"go.etcd.io/etcd/api/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

/**
etcd 控制类，用于从 master 节点中同步所有人 job
*/

var (
	G_jobManager *JobManager
)

type JobManager struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

func InitJobManager() error {
	var (
		err     error
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)

	//配置 etcd 客户端
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDiaTimeOut) * time.Second,
	}

	//连接 etcd
	if client, err = clientv3.New(config); err != nil {
		return err
	}

	//得到 kv 对象来操作数据
	kv = clientv3.NewKV(client)

	//申请租约
	lease = clientv3.NewLease(client)

	//申请任务监听器
	watcher = clientv3.NewWatcher(client)

	//赋值单例
	G_jobManager = &JobManager{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	return nil
}

//监听任务变化
func (jobManager JobManager) WatchJobs() error {
	var (
		getResponse        *clientv3.GetResponse
		kvPair             *mvccpb.KeyValue
		job                *common.Job
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResponse      clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
		err                error
	)
	//1. 得到 /cron/jobs/ 目录下的所有任务，并且获知当前集群的 revision
	if getResponse, err = jobManager.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix()); err != nil {
		return err
	}

	//全量的任务列表
	for _, kvPair = range getResponse.Kvs {
		//反序列化
		if job, err = common.UnmarshalJob(kvPair.Value); err == nil {
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
			//TODO: 把这个 job 同步给 scheduler 调度协程
			G_schedular.PushJobEven(jobEvent)
		}
	}

	//2. 从该 revision 向后监听变化
	go func() {
		watchStartRevision = getResponse.Header.Revision + 1
		//监听 /cron/jobs/ 的后续变化
		watchChan = jobManager.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix(), clientv3.WithRev(watchStartRevision))
		for watchResponse = range watchChan {
			for _, watchEvent = range watchResponse.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //任务保存
					//反序列化 job
					if job, err = common.UnmarshalJob(watchEvent.Kv.Value); err != nil {
						continue
					}
					//TODO: 构建一个 EVENT 更新事件，推送给 scheduler
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
					G_schedular.PushJobEven(jobEvent)

				case mvccpb.DELETE: //任务删除  DELETE /cron/jobs/job1
					//提取出 jobName
					jobName = common.ExtractJobName(common.JOB_SAVE_DIR, string(watchEvent.Kv.Key))
					job = &common.Job{
						Name: jobName,
					}
					//TODO: 推一个删除事件给 scheduler，让它停止执行任务
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)
					G_schedular.PushJobEven(jobEvent)

				}
			}
		}

	}()
	return nil
}

//创建一把分布式锁
func (jobManager *JobManager) CreateJobLock(jobName string) *JobLock {
	return InitJobLock(jobName, jobManager.kv, jobManager.lease)
}

//监听任务的强杀情况
func (jobManager *JobManager) WatchKilledJob() error {
	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobEvent   *common.JobEvent
		jobName    string
		job        *common.Job
		err        error
	)

	//启动一个监听协程去监听 etcd 中 /cron/killer/ 目录的变化
	go func() {
		watchChan = jobManager.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())
		//处理监听事件
		for watchResp = range watchChan {
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //杀死某个任务
					//提取出任务名
					jobName = common.ExtractJobName(common.JOB_KILLER_DIR, string(watchEvent.Kv.Key))
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)
					//推送事件给 scheduler
					G_schedular.PushJobEven(jobEvent)
				case mvccpb.DELETE: // killer 标记过期
					//不作任何处理
				}
			}
		}
	}()

	return err
}
