package common

import "errors"

/**
错误码
*/

var (
	ERR_LOCK_ALREADY_REQUIRED = errors.New("锁已被占用")
)
