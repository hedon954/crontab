package worker

import (
	"context"
	"crontab/common"
	clientv3 "go.etcd.io/etcd/client/v3"
)

/**
分布式锁
*/

type JobLock struct {
	Kv      clientv3.KV
	Lease   clientv3.Lease
	LeaseID clientv3.LeaseID

	JobName   string
	cancelFun context.CancelFunc
	IsLocked  bool
}

//初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) *JobLock {
	return &JobLock{
		Kv:      kv,
		Lease:   lease,
		JobName: jobName,
	}
}

//尝试上锁
func (jobLock *JobLock) TryLock() error {

	var (
		leaseGrantResponse          *clientv3.LeaseGrantResponse
		cancelContext               context.Context
		cancelFunc                  context.CancelFunc
		leaseID                     clientv3.LeaseID
		leaseKeepAliveResponsesChan <-chan *clientv3.LeaseKeepAliveResponse
		txn                         clientv3.Txn
		jobLockPath                 string
		txnResponse                 *clientv3.TxnResponse
		err                         error
	)

	//1. 创建一个 5s 的租约
	if leaseGrantResponse, err = jobLock.Lease.Grant(context.TODO(), 5); err != nil {
		return err
	}
	cancelContext, cancelFunc = context.WithCancel(context.TODO())
	leaseID = leaseGrantResponse.ID

	//2. 自动续租
	if leaseKeepAliveResponsesChan, err = jobLock.Lease.KeepAlive(cancelContext, leaseID); err != nil {
		goto FAIL
	}

	//3. 处理续租应答的协程
	go func() {
		var (
			leaseKeepAliveResponse *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case leaseKeepAliveResponse = <-leaseKeepAliveResponsesChan:
				if leaseKeepAliveResponse == nil {
					goto END
				}
			}
		}
	END:
	}()

	//4. 创建事务 txn
	txn = jobLock.Kv.Txn(context.TODO())
	jobLockPath = common.JOB_LOCK_DIR + jobLock.JobName

	//5. 事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(jobLockPath), "=", 0)).
		Then(clientv3.OpPut(jobLockPath, "", clientv3.WithLease(leaseID))). //抢到
		Else(clientv3.OpGet(jobLockPath))                                   //抢不到
	if txnResponse, err = txn.Commit(); err != nil {
		goto FAIL
	}

	//6. 成功返回 nil，失败释放租约
	if !txnResponse.Succeeded {
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FAIL
	}

	//上锁成功
	jobLock.LeaseID = leaseID
	jobLock.cancelFun = cancelFunc
	jobLock.IsLocked = true
	return nil

FAIL:
	//取消自动续租
	cancelFunc()
	//释放租约
	jobLock.Lease.Revoke(context.TODO(), leaseID)
	return err

}

//释放锁
func (jobLock *JobLock) UnLock() {
	if jobLock.IsLocked {
		//取消自动续租
		jobLock.cancelFun()
		//释放租约
		jobLock.Lease.Revoke(context.TODO(), jobLock.LeaseID)
	}
}
