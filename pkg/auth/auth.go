package auth

import (
	"context"
	"net/http"
	"strings"

	connect_go "github.com/bufbuild/connect-go"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/model"
	"droscher.com/BeerGargoyle/pkg/repository"
)

const devUserEmailHeader = "X-Dev-User-Email"

type UserKey struct{}

type Manager struct {
	conf   *configs.Config
	repo   *repository.Repository
	logger *zap.Logger
	noAuth bool
}

func NewAuthManager(conf *configs.Config, repo *repository.Repository, noAuth bool, logger *zap.Logger) *Manager {
	return &Manager{conf: conf, repo: repo, noAuth: noAuth, logger: logger}
}

func (a *Manager) GrpcAuthInterceptor() connect_go.UnaryInterceptorFunc {
	return func(next connect_go.UnaryFunc) connect_go.UnaryFunc {
		return func(ctx context.Context, req connect_go.AnyRequest) (connect_go.AnyResponse, error) {
			user, err := a.resolveUser(ctx, req.Header())
			if err != nil {
				return nil, err
			}

			return next(context.WithValue(ctx, UserKey{}, user), req)
		}
	}
}

// resolveUser returns the authenticated user, either via JWT or the no-auth dev shortcut.
func (a *Manager) resolveUser(ctx context.Context, header http.Header) (*model.User, error) {
	if a.noAuth {
		return a.resolveDevUser(ctx, header)
	}

	return a.resolveJWTUser(ctx, header)
}

func (a *Manager) resolveDevUser(ctx context.Context, header http.Header) (*model.User, error) {
	email := header.Get(devUserEmailHeader)
	if email == "" {
		return nil, status.Errorf(codes.Unauthenticated, "no-auth mode requires %q header", devUserEmailHeader)
	}

	user, err := a.repo.GetUserFromEmail(ctx, email)
	if err != nil {
		a.logger.Error("no-auth: error looking up dev user", zap.String("email", email), zap.Error(err))

		return nil, status.Errorf(codes.Internal, "no-auth: error looking up dev user")
	}

	if user == nil {
		a.logger.Error("no-auth: dev user not found", zap.String("email", email))

		return nil, status.Errorf(codes.NotFound, "no-auth: user %q not found", email)
	}

	return user, nil
}

func (a *Manager) resolveJWTUser(ctx context.Context, header http.Header) (*model.User, error) {
	accessToken, err := a.extractTokenFromHeader(header)
	if err != nil {
		return nil, err
	}

	keyFunc := func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, status.Errorf(codes.Unauthenticated, "unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(a.conf.Auth.SecretKey), nil
	}

	token, err := jwt.ParseWithClaims(*accessToken, jwt.MapClaims{}, keyFunc)
	if err != nil {
		a.logger.Error("error parsing token", zap.Error(err))

		return nil, status.Errorf(codes.Unauthenticated, "error parsing token: %v", err)
	}

	claims, found := token.Claims.(jwt.MapClaims)
	if !found || !token.Valid {
		a.logger.Error("invalid token", zap.Any("claims", claims))

		return nil, status.Errorf(codes.Unauthenticated, "invalid token")
	}

	a.logger.Debug("claims", zap.Any("claims", claims))

	email, found := claims["email"].(string)
	if !found {
		a.logger.Error("unable to get user id from token", zap.Any("claims", claims))

		return nil, status.Errorf(codes.Unauthenticated, "unable to get user id from token")
	}

	user, err := a.repo.GetUserFromEmail(ctx, email)
	if err != nil {
		a.logger.Error("error authenticating user", zap.Error(err))

		return nil, status.Errorf(codes.Internal, "error authenticating user")
	}

	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	return user, nil
}

func (a *Manager) extractTokenFromHeader(header http.Header) (*string, error) {
	authorization := header.Get("Authorization")
	if len(authorization) == 0 {
		a.logger.Error("No authorization header found")

		return nil, status.Errorf(codes.Unauthenticated, "authorization header not found")
	}

	prefix := "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		prefix = "bearer "
	}

	token, found := strings.CutPrefix(authorization, prefix)
	if !found {
		return nil, status.Errorf(codes.Unauthenticated, "authorization format must be Bearer {token}")
	}

	return &token, nil
}
