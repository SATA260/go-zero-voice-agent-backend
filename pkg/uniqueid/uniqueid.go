package uniqueid

import (
	"github.com/sony/sonyflake/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

var flake *sonyflake.Sonyflake

func init() {
	flake, _ = sonyflake.New(sonyflake.Settings{})
}

func GenId() int64 {

	id, err := flake.NextID()
	if err != nil {
		logx.Severef("flake NextID failed with %s \n", err)
		panic(err)
	}

	return int64(id)
}
