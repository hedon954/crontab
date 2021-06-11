package master

import (
	"context"
	"crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

/**
	日志管理器，主要负责与 mongodb 的交互
 */

var(
	G_logManager *LogManager
)

type LogManager struct {
	client *mongo.Client
	logCollection *mongo.Collection
}



func InitLogManager() error {
	var(
		client *mongo.Client
		ctx context.Context
		err error
	)

	//建立与 mongodb 的连接
	ctx, _ = context.WithTimeout(context.TODO(), time.Duration(G_config.MongodbConnectTimeOut) * time.Millisecond)
	if client, err = mongo.Connect(ctx, options.Client().ApplyURI(G_config.MongodbURI)); err != nil {
		return err
	}

	//赋值单例
	G_logManager = &LogManager{
		client: client,
		logCollection: client.Database("cron").Collection("log"),
	}

	return nil

}

//查看任务日志
func (logManager *LogManager) ListLog(jobName string, skip int, limit int) (logArr []*common.JobLog, err error) {
	var(
		jobLogFilter *common.JobLogFilter
		jobLogSorter *common.JobLogSorter
		cursor *mongo.Cursor
		jobLog *common.JobLog
	)
	//过滤条件
	jobLogFilter = &common.JobLogFilter{
		JobName: jobName,
	}

	//按照任务开始时间倒排
	jobLogSorter = &common.JobLogSorter{
		SortOrder: -1,
	}

	skipP := int64(skip)
	limitP := int64(limit)

	//查询
	if cursor, err = logManager.logCollection.Find(context.TODO(), jobLogFilter,
		&options.FindOptions{
			Sort:  jobLogSorter,
			Skip:  &skipP,
			Limit: &limitP,
		}); err != nil {
		return nil,err
	}
	defer cursor.Close(context.TODO())

	logArr = make([]*common.JobLog, 0)

	//遍历结果
	for cursor.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		//反序列化
		if err = cursor.Decode(jobLog); err != nil {
			continue	//日志不合法就忽略
		}
		logArr = append(logArr, jobLog)
	}

	return logArr, nil
}


