package consts

const (
	RegisterVerifyCodeCachePrefix = "usercenter:reg:verify_code:"
)

func GetRegisterVerifyCodeCacheKey(email string) string {
	return RegisterVerifyCodeCachePrefix + email
}