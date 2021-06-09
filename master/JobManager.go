package master

import (
	"context"
	"crontab/common"
	"encoding/json"
	"go.etcd.io/etcd/api/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

//单例
var (
	G_jobManager *JobManager
)

//定时任务的管理器
type JobManager struct {
	client * clientv3.Client
	kv clientv3.KV
	lease clientv3.Lease
}

//初始化定时任务管理器
func InitJobManager() error {

	var(
		err error
		config clientv3.Config
		client *clientv3.Client
		kv clientv3.KV
		lease clientv3.Lease
	)

	//配置 etcd 客户端
	config = clientv3.Config{
		Endpoints: G_config.EtcdEndPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeOut) * time.Millisecond,
	}

	//连接
	if client, err = clientv3.New(config); err != nil {
		return err
	}

	//获取一个 kv 对象来操作数据
	kv = clientv3.NewKV(client)

	//获取租约
	lease = clientv3.NewLease(client)

	//赋值单例
	G_jobManager = &JobManager{
		client: client,
		kv: kv,
		lease: lease,
	}

	return nil
}


// SaveJob 保存定时任务
func (jobManager *JobManager) SaveJob(job *common.Job) (oldJob *common.Job, err error) {

	var(
		jobKey string
		jobValue []byte
		putResponse *clientv3.PutResponse
		oldJobObj common.Job
	)

	//1. 确定任务 key
	jobKey = common.JOB_SAVE_DIR + job.Name

	//2. 确定任务 value
	if jobValue,err = json.Marshal(job); err != nil {
		return
	}

	//3. 保存到 etcd
	if putResponse, err = jobManager.kv.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV()); err != nil {
		return
	}

	//4. 如果是更新，返回旧值
	if putResponse.PrevKv != nil {
		//对旧值进行反序列化
		if err = json.Unmarshal(putResponse.PrevKv.Value, &oldJobObj); err != nil {
			err = nil
		}else{
			oldJob = &oldJobObj
		}
	}

	return
}

// DeleteJob 删除任务
func (jobManager *JobManager) DeleteJob(jobName string)(oldJob *common.Job, err error)  {
	var(
		jobKey string
		oldJobObj common.Job
		deleteResponse *clientv3.DeleteResponse

	)

	//1. 构建 jobKey
	jobKey = common.JOB_SAVE_DIR + jobName

	//2. 删除 Job
	if deleteResponse, err = jobManager.kv.Delete(context.TODO(), jobKey, clientv3.WithPrevKV()); err != nil {
		return nil, err
	}

	//3. 检查是否有旧值
	if len(deleteResponse.PrevKvs) != 0 {
		if err = json.Unmarshal(deleteResponse.PrevKvs[0].Value, &oldJobObj); err != nil {
			err = nil
		}else{
			oldJob = &oldJobObj
		}
	}

	return oldJob, err
}

// ListJob 列举出所有任务
func (jobManager *JobManager) ListJob() (jobs []*common.Job, err error) {
	var(
		dirKey string
		getResponse *clientv3.GetResponse
		job *common.Job
		kvPair *mvccpb.KeyValue
	)

	dirKey = common.JOB_SAVE_DIR

	if getResponse, err = G_jobManager.kv.Get(context.TODO(), dirKey, clientv3.WithPrefix()); err != nil {
		return nil, err
	}

	//初始化 slice 空间
	jobs = make([]*common.Job, 0)

	//遍历所有任务，进行反序列化，并加到slice中
	kvs := getResponse.Kvs
	for i := 0; i < len(kvs); i++ {
		kvPair = kvs[i]
		job = &common.Job{}
		if err = json.Unmarshal(kvPair.Value, job); err != nil {
			continue
		}
		jobs = append(jobs,job)
	}

	return jobs, nil
}

// KillJob 强制终止某个任务
func (jobManager JobManager) KillJob(jobName string) (err error) {
	var(
		killerKey string
		leaseGrantResponse *clientv3.LeaseGrantResponse
		leaseID clientv3.LeaseID
	)

	//1. 构建终止任务的 key：/cron/killer/任务名
	killerKey = common.JOB_KILLER_DIR + jobName

	//2. 通知 worker 去杀死任务：让 worker 监听到 etcd 的一次 put 操作
	if leaseGrantResponse, err = G_jobManager.lease.Grant(context.TODO(), 1); err != nil {
		return err
	}
	leaseID = leaseGrantResponse.ID
	if _, err = G_jobManager.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseID)); err != nil {
		return err
	}

	return nil

}