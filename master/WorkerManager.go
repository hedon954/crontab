package master

import (
	"context"
	"crontab/common"
	"go.etcd.io/etcd/api/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

/**
worker 管理类

提供服务发现功能
*/

var (
	G_workerManager *WorkerManager
)

type WorkerManager struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

func InitWorkerManager() error {
	var (
		err    error
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)

	//客户端配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeOut) * time.Millisecond,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return err
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	//赋值单例
	G_workerManager = &WorkerManager{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return nil
}

//获取健康 worker 集群列表
func (workerManager *WorkerManager) ListWorkers() (workerAddr []string, err error) {
	var (
		findKeyPrefix string
		getResponse   *clientv3.GetResponse
		kv            *mvccpb.KeyValue
	)

	//初始化数组
	workerAddr = make([]string, 0)

	//构建查询前缀
	findKeyPrefix = common.JOB_WORKERS_DIR

	//查询
	if getResponse, err = workerManager.kv.Get(context.TODO(), findKeyPrefix, clientv3.WithPrefix()); err != nil {
		return nil, err
	}

	//封装
	for _, kv = range getResponse.Kvs {
		//解析出 key 中的 ip 地址 (/cron/workers/255.255.255.255)
		workerAddr = append(workerAddr, common.ExtractWorkerIp(string(kv.Key)))
	}

	return workerAddr, nil

}
