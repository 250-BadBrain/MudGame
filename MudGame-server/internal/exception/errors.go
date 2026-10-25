package exception

import "errors"

var (
	ErrUnknownCommand = errors.New("未知指令")

	ErrUserNotFound   = errors.New("用户不存在")
	ErrWrongPassword  = errors.New("密码错误")
	ErrPlayerNotFound = errors.New("角色不存在")
	ErrRoomNotFound   = errors.New("房间不存在")

	ErrPlayerNotSelected = errors.New("未选择角色，请先登录并选择角色")
)
