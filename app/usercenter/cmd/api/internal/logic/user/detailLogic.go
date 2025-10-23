// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"strconv"

	"go-zero-voice-agent/app/usercenter/cmd/api/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/api/internal/types"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get rpc info
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	if req.UserId == "" {
		return nil, errors.New("user id is empty")
	}

	userId, _ := strconv.ParseInt(req.UserId, 10, 64)
	userInfo, err := l.svcCtx.UsercenterRpc.GetUserInfo(l.ctx, &pb.GetUserInfoReq{
		Id: userId,
	})
	if err != nil {
		return nil, err
	}

	return &types.UserInfoResp{
		Id:       userInfo.User.Id,
		Email:    userInfo.User.Email,
		NickName: userInfo.User.Nickname,
		Sex:      userInfo.User.Sex,
		Avatar:   userInfo.User.Avatar,
		Info:     userInfo.User.Info,
	}, nil
}
