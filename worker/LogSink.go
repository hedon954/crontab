package worker

import (
	"context"
	"crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

/**
负责 mongodb 日志的读写
*/

var (
	//单例
	G_logSink *LogSink
)

type LogSink struct {
	Client         *mongo.Client
	LogCollection  *mongo.Collection
	LogChan        chan *common.JobLog
	AutoCommitChan chan *common.LogBatch
}

//初始化 LogSink
func InitLogSink() error {
	var (
		client *mongo.Client
		ctx    context.Context
		err    error
	)

	//建立与 mongodb 的连接
	ctx, _ = context.WithTimeout(context.Background(), time.Duration(G_config.MongodbConnectTimeOut)*time.Millisecond)
	if client, err = mongo.Connect(ctx, options.Client().ApplyURI(G_config.MongodbURI)); err != nil {
		return err
	}

	//赋值单例
	G_logSink = &LogSink{
		Client:         client,
		LogCollection:  client.Database("cron").Collection("log"),
		LogChan:        make(chan *common.JobLog, 1024),
		AutoCommitChan: make(chan *common.LogBatch, 1024),
	}

	//启动日志协程
	go G_logSink.writeLoop()

	return nil
}

//启动一个 mongodb 日志存储协程
func (logSink *LogSink) writeLoop() {
	var (
		log             *common.JobLog
		logBatch        *common.LogBatch //当前的批次
		logTimeOutBacth *common.LogBatch //超时批次
		commitTimer     *time.Timer      //超时未满自动提交
	)

	for {
		select {
		case log = <-logSink.LogChan:
			//批次把 log 写到 mongodb 中
			if logBatch == nil {
				logBatch = &common.LogBatch{}
				commitTimer = time.AfterFunc(time.Duration(G_config.JobLogCommitTimeOut)*time.Millisecond,
					//TODO: ???
					func(batch *common.LogBatch) func() {
						return func() {
							logSink.AutoCommitChan <- batch
						}
					}(logBatch))
			}
			logBatch.Logs = append(logBatch.Logs, log)

			//批次满了，就发送到 mongodb
			if len(logBatch.Logs) >= G_config.JogLogBatchSize {
				logSink.saveLogs(logBatch)
				//清空
				logBatch = nil
				//取消定时器
				commitTimer.Stop()
			}
		case logTimeOutBacth = <-logSink.AutoCommitChan:
			//判断过期批次是否仍然是当前批次，不是说明已经被提交了，跳过
			if logTimeOutBacth != logBatch {
				continue
			}
			//写入到 mongodb
			logSink.saveLogs(logTimeOutBacth)
			logBatch = nil
		}

	}
}

//批量写入日志
func (logSink *LogSink) saveLogs(batch *common.LogBatch) {
	logSink.LogCollection.InsertMany(context.TODO(), batch.Logs)
}

//发送日志
func (logSink *LogSink) AppendLog(jobLog *common.JobLog) {
	select {
	case logSink.LogChan <- jobLog:
	default:
		//队列满了就丢弃
	}
}
