package xerr

var msg map[uint32]string

func init() {
	msg = make(map[uint32]string)
	// 全局错误
	msg[OK] = "SUCCESS"
	msg[SERVER_COMMON_ERROR] = "服务器开小差啦,稍后再来试一试"
	msg[REQUEST_PARAM_ERROR] = "参数错误"
	msg[TOKEN_EXPIRE_ERROR] = "token失效，请重新登陆"
	msg[TOKEN_GENERATE_ERROR] = "生成token失败"
	msg[DB_ERROR] = "数据库繁忙,请稍后再试"
	msg[DB_UPDATE_AFFECTED_ZERO_ERROR] = "更新数据影响行数为0"

	// 用户模块
	msg[USER_NOT_FOUND_ERROR] = "用户不存在"
	msg[USER_PASSWORD_ERROR] = "用户名或密码错误"
	msg[USER_REGISTER_ERROR] = "用户注册失败"
	msg[USER_PERMISSION_DENIED_ERROR] = "用户无权限"
	msg[USER_ALREADY_EXISTS_ERROR] = "用户已存在"
}

func MapErrMsg(errcode uint32) string {
	if message, ok := msg[errcode]; ok {
		return message
	} else {
		return "服务器开小差啦,请稍后再试"
	}
}

func IsCodeErr(errCode uint32) bool {
	if _, ok := msg[errCode]; ok {
		return true
	} else {
		return false
	}
}
