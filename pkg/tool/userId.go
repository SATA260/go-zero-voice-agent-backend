package tool

import (
	"net/http"

	"github.com/pkg/errors"
)

func GetUserIdFromHeader(r *http.Request) (string, error) {
	userId := r.Header.Get("X-User-Id")
	if userId == "" {
		return "", errors.New("missing X-User-Id header")
	}
	return userId, nil
}