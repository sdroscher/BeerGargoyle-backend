package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/connect-go"
	"go.openly.dev/pointy"
	"go.uber.org/zap"

	api "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/grpc/api/v1/apiv1connect"
	"droscher.com/BeerGargoyle/pkg/repository"
)

var ErrUserNotFound = errors.New("user not found")

type UserServer struct {
	apiv1connect.UnimplementedUserServiceHandler
	repository repository.UserRepository
	logger     *zap.Logger
}

func NewUserServer(repository repository.UserRepository, logger *zap.Logger) *UserServer {
	return &UserServer{repository: repository, logger: logger}
}

func (u *UserServer) AddUser(ctx context.Context, request *connect.Request[api.AddUserRequest]) (*connect.Response[api.AddUserResponse], error) {
	if request.Msg.GetEmail() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w: email is required", ErrInvalidInput))
	}

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
