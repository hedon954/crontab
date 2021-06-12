package worker

import (
	"context"
	"crontab/common"
	clientv3 "go.etcd.io/etcd/client/v3"
	"net"
	"time"
)

/**
负责 worker 的服务注册功能

注册结点到etcd  /cron/workers/IP地址
*/

var (
	G_register *Register
)

type Register struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIP string
}

//	初始化注册工具
func InitRegister() error {

	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		localIP string
		err     error
	)

	//客户端配置
	config = clientv3.Config{
		Endpoints:   G_config.EtcdEndpoints,
		DialTimeout: time.Duration(G_config.EtcdDiaTimeOut) * time.Millisecond,
	}

	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return err
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	if localIP, err = getLocalIP(); err != nil {
		return err
	}

	//赋值单例
	G_register = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localIP,
	}

	//注册
	go G_register.keepOnline()

	return nil
}

//	读取本地网卡 IP
func getLocalIP() (ipv4 string, err error) {

	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet
		isIpNet bool
	)

	//获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return "", err
	}

	//取第一个非 localhost 的网卡
	for _, addr = range addrs {
		//解析出其中的 ipv4
		//需要跳过非 IP 地址和环回地址
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			//跳过 ipv6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				return ipv4, nil
			}
		}
	}

	return "", common.ERR_NO_LOCAL_IP_FOUND
}

//自动注册到 /cron/workers/IP地址，并且自动续租
func (register *Register) keepOnline() {
	var (
		regKey                 string
		cancelCtx              context.Context
		cancelFunc             context.CancelFunc
		leaseGrantResponse     *clientv3.LeaseGrantResponse
		keepAliveResponsesChan <-chan *clientv3.LeaseKeepAliveResponse
		keepAliveResponse      *clientv3.LeaseKeepAliveResponse
		err                    error
	)

	for {
		//注册路径
		regKey = common.JOB_WORKERS_DIR + register.localIP

		cancelFunc = nil

		//创建租约
		if leaseGrantResponse, err = register.lease.Grant(context.TODO(), 3); err != nil {
			goto RETRY
		}

		//自动续租
		if keepAliveResponsesChan, err = register.lease.KeepAlive(context.TODO(), leaseGrantResponse.ID); err != nil {
			goto RETRY
		}

		cancelCtx, cancelFunc = context.WithCancel(context.TODO())

		//注册到 etcd
		if _, err = register.kv.Put(cancelCtx, regKey, "", clientv3.WithLease(leaseGrantResponse.ID)); err != nil {
			goto RETRY
		}

		//处理续租应答
		select {
		case keepAliveResponse = <-keepAliveResponsesChan:
			if keepAliveResponse == nil { //续租失败(一般是网络不通)
				goto RETRY
			}
		}

		//重试
	RETRY:
		time.Sleep(1 * time.Second)
		//取消自动续租
		if cancelFunc != nil {
			cancelFunc()
		}
	}

}
