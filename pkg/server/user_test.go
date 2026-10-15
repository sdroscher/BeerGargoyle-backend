package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bufbuild/connect-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.openly.dev/pointy"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"

	apiv1 "droscher.com/BeerGargoyle/generated/grpc/api/v1"
	"droscher.com/BeerGargoyle/generated/mocks"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/server"
)

type UserServerTestSuite struct {
	suite.Suite
	userRepo     *mocks.UserRepository
	observedLogs *observer.ObservedLogs
	service      *server.UserServer
}

func TestUserServerTestSuite(t *testing.T) {
	suite.Run(t, new(UserServerTestSuite))
}

func (suite *UserServerTestSuite) SetupTest() {
	suite.userRepo = mocks.NewUserRepository(suite.T())
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	suite.observedLogs = observedLogs
	suite.service = server.NewUserServer(suite.userRepo, zap.New(observedZapCore))
}

func (suite *UserServerTestSuite) TestAddUser_Success() {
	ctx := context.Background()
	userUUID := uuid.New()
	suite.userRepo.EXPECT().AddUser(ctx, bobName, bobEmail, pointy.String(bobUntappd)).
		Return(&model.User{Model: gorm.Model{ID: 1}, UUID: userUUID, Username: bobName, Email: bobEmail}, nil)

	result, err := suite.service.AddUser(ctx, connect.NewRequest(&apiv1.AddUserRequest{
		Name: bobName, Email: bobEmail, UntappedUsername: bobUntappd,
	}))

	suite.Require().NoError(err)
	user := result.Msg.GetUser()
	suite.Equal(userUUID.String(), user.GetId())
	suite.Equal(bobName, user.GetUserName())
	suite.Equal(bobEmail, user.GetEmail())
	suite.Equal(bobUntappd, user.GetUntappedUsername())
}

func (suite *UserServerTestSuite) TestAddUser_EmptyUntappdUsernameIsStoredAsNil() {
	ctx := context.Background()
	suite.userRepo.EXPECT().AddUser(ctx, bobName, bobEmail, mock.MatchedBy(func(untappd *string) bool { return untappd == nil })).
		Return(&model.User{UUID: uuid.New(), Username: bobName, Email: bobEmail}, nil)

	result, err := suite.service.AddUser(ctx, connect.NewRequest(&apiv1.AddUserRequest{Name: bobName, Email: bobEmail}))

	suite.Require().NoError(err)
	suite.Nil(result.Msg.GetUser().UntappedUsername)
}

func (suite *UserServerTestSuite) TestAddUser_EmptyEmailIsInvalidArgument() {
	result, err := suite.service.AddUser(context.Background(), connect.NewRequest(&apiv1.AddUserRequest{Name: bobName}))

	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *UserServerTestSuite) TestAddUser_WhitespaceOnlyEmailIsInvalidArgument() {
	result, err := suite.service.AddUser(context.Background(), connect.NewRequest(&apiv1.AddUserRequest{Name: bobName, Email: "   "}))

	suite.Require().ErrorIs(err, server.ErrInvalidInput)
	suite.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *UserServerTestSuite) TestAddUser_RepositoryError() {
	ctx := context.Background()
	suite.userRepo.EXPECT().AddUser(ctx, bobName, bobEmail, mock.Anything).Return(nil, errors.New("duplicate email"))

	result, err := suite.service.AddUser(ctx, connect.NewRequest(&apiv1.AddUserRequest{Name: bobName, Email: bobEmail}))

	suite.Require().ErrorContains(err, "duplicate email")
	suite.Nil(result)
}

func (suite *UserServerTestSuite) TestGetUserByEmail_Success() {
	ctx := context.Background()
	userUUID := uuid.New()
	suite.userRepo.EXPECT().GetUserFromEmail(ctx, bobEmail).Return(&model.User{
		UUID: userUUID, Username: bobName, Email: bobEmail, UntappdUserName: pointy.String(bobUntappd),
	}, nil)

	result, err := suite.service.GetUserByEmail(ctx, connect.NewRequest(&apiv1.GetUserByEmailRequest{Email: bobEmail}))

	suite.Require().NoError(err)
	user := result.Msg.GetUser()
	suite.Equal(userUUID.String(), user.GetId())
	suite.Equal(bobName, user.GetUserName())
	suite.Equal(bobUntappd, user.GetUntappedUsername())
}

func (suite *UserServerTestSuite) TestGetUserByEmail_NoUntappdUsername() {
	ctx := context.Background()
	suite.userRepo.EXPECT().GetUserFromEmail(ctx, bobEmail).
		Return(&model.User{UUID: uuid.New(), Username: bobName, Email: bobEmail}, nil)

	result, err := suite.service.GetUserByEmail(ctx, connect.NewRequest(&apiv1.GetUserByEmailRequest{Email: bobEmail}))

	suite.Require().NoError(err)
	suite.Nil(result.Msg.GetUser().UntappedUsername)
}

func (suite *UserServerTestSuite) TestGetUserByEmail_NotFound() {
	ctx := context.Background()
	suite.userRepo.EXPECT().GetUserFromEmail(ctx, nobodyEmail).Return(nil, gorm.ErrRecordNotFound)

	result, err := suite.service.GetUserByEmail(ctx, connect.NewRequest(&apiv1.GetUserByEmailRequest{Email: nobodyEmail}))

	suite.Require().ErrorIs(err, server.ErrUserNotFound)
	suite.Equal(connect.CodeNotFound, connect.CodeOf(err))
	suite.Nil(result)
}

func (suite *UserServerTestSuite) TestGetUserByEmail_OtherErrorsAreNotReportedAsNotFound() {
	ctx := context.Background()
	suite.userRepo.EXPECT().GetUserFromEmail(ctx, bobEmail).Return(nil, errors.New(dbDownMessage))

	result, err := suite.service.GetUserByEmail(ctx, connect.NewRequest(&apiv1.GetUserByEmailRequest{Email: bobEmail}))

	suite.Require().ErrorContains(err, dbDownMessage)
	suite.Require().NotErrorIs(err, server.ErrUserNotFound)
	suite.NotEqual(connect.CodeNotFound, connect.CodeOf(err))
	suite.Nil(result)
}
