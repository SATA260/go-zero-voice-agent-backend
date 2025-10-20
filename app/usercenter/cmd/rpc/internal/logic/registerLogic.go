package logic

import (
	"context"

	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/consts"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/pb"
	"go-zero-voice-agent/app/usercenter/cmd/rpc/usercenter"
	"go-zero-voice-agent/app/usercenter/model"
	"go-zero-voice-agent/pkg/tool"
	"go-zero-voice-agent/pkg/xerr"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrUserAlreadyRegisterError = xerr.NewErrMsg("user has been registered")

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterResp, error) {
	user, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, in.Email)
	if err != nil && err != model.ErrNotFound {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DB_ERROR), "mobile:%s, err:%v", in.Email, err)
	}
	if user != nil {
		return nil, errors.Wrapf(ErrUserAlreadyRegisterError, "Register user exists email:%s, err:%v", in.Email, err)
	}

	// 验证验证码
	verifyCodeKey := consts.GetRegisterVerifyCodeCacheKey(in.Email)
	cachedCode, err := l.svcCtx.RedisClient.Get(verifyCodeKey)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DB_ERROR), "get verify code from redis err:%v", err)
	}
	if cachedCode != in.VerifyCode {
		return nil, errors.Wrapf(xerr.NewErrMsg("verify code error"), "email:%s, input code:%s, cache code:%s", in.Email, in.VerifyCode, cachedCode)
	}

	// 创建用户
	var userId int64
	if err := l.svcCtx.UserModel.Trans(l.ctx, func(context context.Context, session sqlx.Session) error {
		user := new(model.User)
		user.Email = in.Email
		if len(in.Nickname) == 0 {
			user.Nickname = tool.Krand(8, tool.KC_RAND_KIND_ALL)
		}
		if len(in.Password) > 0 {
			user.Password = tool.MD5HashWithSalt(in.Password, model.UserPassWordSalt)
		}

		insertResult, err := l.svcCtx.UserModel.Insert(context, session, user)
		if err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DB_ERROR), "Register db user Insert err:%v,user:%+v", err, user)
		}
		lastId, err := insertResult.LastInsertId()
		if err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DB_ERROR), "Register db user insertResult.LastInsertId err:%v,user:%+v", err, user)
		}
		userId = lastId

		userAuth := new(model.UserAuth)
		userAuth.UserId = lastId
		userAuth.AuthKey = in.AuthKey
		userAuth.AuthType = in.AuthType
		if _, err := l.svcCtx.UserAuthModel.Insert(context, session, userAuth); err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DB_ERROR), "Register db user_auth Insert err:%v,userAuth:%v", err, userAuth)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	generateTokenLogic := NewGenerateTokenLogic(l.ctx, l.svcCtx)
	tokenResp, err := generateTokenLogic.GenerateToken(&usercenter.GenerateTokenReq{
		UserId: userId,
	})
	if err != nil {
		return nil, errors.Wrapf(ErrGenerateTokenError, "GenerateToken userId : %d", userId)
	}

	return &usercenter.RegisterResp{
		AccessToken:  tokenResp.AccessToken,
		AccessExpire: tokenResp.AccessExpire,
		RefreshAfter: tokenResp.RefreshAfter,
	}, nil
}
