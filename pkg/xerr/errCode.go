package xerr

// 成功返回
const OK uint32 = 200

/**(前3位代表业务,后三位代表具体功能)**/

// 全局错误码
const (
	SERVER_COMMON_ERROR           uint32 = 100001 // 服务器通用错误
	REQUEST_PARAM_ERROR           uint32 = 100002 // 请求参数错误
	TOKEN_EXPIRE_ERROR            uint32 = 100003 // Token 过期
	TOKEN_GENERATE_ERROR          uint32 = 100004 // Token 生成失败
	DB_ERROR                      uint32 = 100005 // 数据库错误
	DB_UPDATE_AFFECTED_ZERO_ERROR uint32 = 100006 // 数据库更新影响行数为0
)

// 用户模块
const (
	USER_NOT_FOUND_ERROR         uint32 = 200001 // 用户未找到
	USER_PASSWORD_ERROR          uint32 = 200002 // 用户密码错误
	USER_REGISTER_ERROR          uint32 = 200003 // 用户注册失败
	USER_PERMISSION_DENIED_ERROR uint32 = 200004 // 用户无权限
	USER_ALREADY_EXISTS_ERROR    uint32 = 200005 // 用户已存在
)
