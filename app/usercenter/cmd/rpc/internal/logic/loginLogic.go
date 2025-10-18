package logic

import (
	"context"

	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"
	"go-zero-voice-agent/app/usercenter/model"
	"go-zero-voice-agent/pkg/tool"
	"go-zero-voice-agent/pkg/xerr"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

var ErrUserNoExistsError = xerr.NewErrMsg("用户不存在")

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
	var userId int64
	var err error

	// 根据不同的登录类型，进行不同的登录处理
	switch in.AuthType {
	case model.UserAuthTypeEmail:
		userId, err = l.loginByEmail(in.AuthKey, in.Password)
	default:
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if err != nil {
		return nil, err
	}

	// 生成token
	generateTokenLogic := NewGenerateTokenLogic(l.ctx, l.svcCtx)
	tokenResp, err := generateTokenLogic.GenerateToken(&pb.GenerateTokenReq{
		UserId:   userId,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "生成用户登录token失败, userId: %d", userId)
	}
	
	// 返回登录结果
	logx.Infof("用户登录成功, userId: %d, token: %s, AccessExpire: %d, RefreshAfter: %d", userId, tokenResp.AccessToken, tokenResp.AccessExpire, tokenResp.RefreshAfter)


	return &pb.LoginResp{
		AccessToken: tokenResp.AccessToken,
		AccessExpire: tokenResp.AccessExpire,
		RefreshAfter: tokenResp.RefreshAfter,
	}, nil
}

func (l *LoginLogic) loginByEmail(email string, password string) (int64, error) {
	user, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, email)
	if err != nil && err != model.ErrNotFound {
		return 0, errors.Wrapf(xerr.NewErrCode(xerr.DB_ERROR), "根据邮箱查询用户信息失败, email: %s, err: %v", email, err)
	}
	if user == nil {
		return 0, errors.Wrapf(ErrUserNoExistsError, "用户不存在, email: %s", email)
	}
	if !(tool.MD5HashWithSalt(password, model.UserPassWordSalt) == user.Password) {
		return 0, xerr.NewErrMsg("用户密码错误")
	}

	return user.Id, nil
}