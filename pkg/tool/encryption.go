package tool

import (
	"crypto/md5"
	"fmt"
	"io"
)

// MD5Hash md5加密
func MD5Hash(text string) string {
	h := md5.New()
	_, err := io.WriteString(h, text)
	if err != nil {
		panic(err)
	}
	arr := h.Sum(nil)
	return fmt.Sprintf("%x", arr)
}

// MD5HashWithSalt md5加盐加密
func MD5HashWithSalt(text, salt string) string {
	h := md5.New()
	_, err := io.WriteString(h, text+salt)
	if err != nil {
		panic(err)
	}
	arr := h.Sum(nil)
	return fmt.Sprintf("%x", arr)
}

// MD5HashBytes md5加密字节数组
func MD5HashBytes(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}