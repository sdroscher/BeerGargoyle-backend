package server

import (
	"context"
	"errors"

	"github.com/bufbuild/connect-go"
	"go.openly.dev/pointy"
	"go.uber.org/zap"

	"droscher.com/BeerGargoyle/pkg/repository"
	api "droscher.com/BeerGargoyle/pkg/server/grpc/api/v1"
	"droscher.com/BeerGargoyle/pkg/server/grpc/api/v1/apiv1connect"
)

var ErrUserNotFound = errors.New("user not found")

type UserServer struct {
	apiv1connect.UnimplementedUserServiceHandler
	repository *repository.Repository
	logger     *zap.Logger
}

func NewUserServer(repository *repository.Repository, logger *zap.Logger) *UserServer {
	return &UserServer{repository: repository, logger: logger}
}

func (u *UserServer) AddUser(ctx context.Context, request *connect.Request[api.AddUserRequest]) (*connect.Response[api.AddUserResponse], error) {
	var untappdUserName *string

	if len(request.Msg.GetUntappedUsername()) > 0 {
		untappdUserName = pointy.String(request.Msg.GetUntappedUsername())
	}

	user, err := u.repository.AddUser(ctx, request.Msg.GetName(), request.Msg.GetEmail(), untappdUserName)
	if err != nil {
		return nil, err
	}

	grpcUser := api.User{
		Id:               user.UUID.String(),
		UserName:         user.Username,
		Email:            user.Email,
		UntappedUsername: untappdUserName,
	}

	return connect.NewResponse(&api.AddUserResponse{User: &grpcUser}), nil
}

func (u *UserServer) GetUserByEmail(ctx context.Context, request *connect.Request[api.GetUserByEmailRequest]) (*connect.Response[api.GetUserByEmailResponse], error) {
	user, err := u.repository.GetUserFromEmail(ctx, request.Msg.GetEmail())
	if err != nil {
		return nil, err
	}

	grpcUser := api.User{
		Id:               user.UUID.String(),
		UserName:         user.Username,
		Email:            user.Email,
		UntappedUsername: user.UntappdUserName,
	}

	return connect.NewResponse(&api.GetUserByEmailResponse{User: &grpcUser}), nil
}
