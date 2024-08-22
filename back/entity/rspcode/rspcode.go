package rspcode

import (
	"errors"
	"fmt"
)

type Code int

// 返回码
const (
	CodeSuccess Code = 1 // 成功
	CodeFailure Code = 0 // 失败

	CodeDatabaseError Code = 2 // 数据库访问出错 10
)

// 返回码提示信息
var (
	errMap = map[Code]string{
		CodeSuccess:       "success",
		CodeFailure:       "failure",
		CodeDatabaseError: "database error",
	}
)

func NewError(code Code, text string) error {
	message := "null"
	if text != "" {
		message = text
	}
	return errors.New(fmt.Sprintf("response code:%v-%v\n,message:%v", code, errMap[code], message))
}
