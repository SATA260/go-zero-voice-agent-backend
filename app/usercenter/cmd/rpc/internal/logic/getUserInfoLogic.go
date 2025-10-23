package logic

import (
	"context"

	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserInfoLogic) GetUserInfo(in *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {

	user, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to get user info")
	}
	
	return &pb.GetUserInfoResp{
		User: &pb.User{
			Id:       user.Id,
			Email:    user.Email,
			Nickname: user.Nickname,
			Sex:      user.Sex,
			Avatar:   user.Avatar,
			Info:     user.Info,
		},
	}, nil
}
