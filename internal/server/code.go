package server

import "fmt"

type Code int

const (
	CodeSuccess Code = iota
	CodeGroupNameExists
)

const (
	CodeErrorStart = iota + 100
	CodeProtocol
	CodeMissArgs
	CodeInvalidArgs
	CodeInternalError
	CodeVerifyFailed
	CodeInvalidToken
	CodeNeedAuth
	CodeDisabled
)

func (c Code) String() string {
	switch c {
	case CodeSuccess:
		return "成功"
	case CodeProtocol:
		return "通信出错"
	case CodeMissArgs:
		return "缺少参数"
	case CodeInvalidArgs:
		return "参数非法"
	case CodeInternalError:
		return "内部错误"
	case CodeVerifyFailed:
		return "验证失败"
	case CodeInvalidToken:
		return "凭证非法"
	case CodeNeedAuth:
		return "需要授权"
	case CodeDisabled:
		return "被禁止"
	}

	return fmt.Sprintf("未知错误%d", c)
}

func CodeToMessage(code Code, msg string) string {
	codeMsg := code.String()

	if msg != "" {
		codeMsg += ":" + msg
	}

	return codeMsg
}
