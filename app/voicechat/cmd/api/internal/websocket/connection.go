package websocket

import (
	"net/http"
	wsTool "github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

var upgrader = wsTool.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewConnection(w http.ResponseWriter, r *http.Request) (*wsTool.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, errors.Errorf("Fail to upgrade http to websocket, %v", err)
	}

	return conn, nil
}