package consts

const (
	RegisterVerifyCodeCachePrefix = "reg:verify_code:"
)

func GetRegisterVerifyCodeCacheKey(email string) string {
	return RegisterVerifyCodeCachePrefix + email
}