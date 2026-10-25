package engine

import "errors"

var (
	errReconnectTokenInvalid  = errors.New("登录状态已失效，请重新登录")
	errReconnectPlayerInvalid = errors.New("角色信息无效，请重新选择角色")
)
