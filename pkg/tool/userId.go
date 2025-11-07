package tool

import (
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

func GetUserIdFromHeader(r *http.Request) (string, error) {
	userId := r.Header.Get("X-User-Id")
	if userId == "" {
		return "", errors.New("missing X-User-Id header")
	}
	return userId, nil
}

func GetUserIdInt64FromHeader(r *http.Request) (int64, error) {
	userIdStr, err := GetUserIdFromHeader(r)
	if err != nil {
		return 0, err
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "invalid X-User-Id header")
	}
	return userId, nil
}