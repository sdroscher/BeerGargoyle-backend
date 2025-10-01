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
	"droscher.com/BeerGargoyle/pkg/repository"
)

type UserKey struct{}

type Manager struct {
	conf   *configs.Config
	repo   *repository.Repository
	logger *zap.Logger
}

func NewAuthManager(conf *configs.Config, repo *repository.Repository, logger *zap.Logger) *Manager {
	return &Manager{conf: conf, repo: repo, logger: logger}
}

func (a *Manager) GrpcAuthInterceptor() connect_go.UnaryInterceptorFunc {
	return func(next connect_go.UnaryFunc) connect_go.UnaryFunc {
		return func(ctx context.Context, req connect_go.AnyRequest) (connect_go.AnyResponse, error) {
			keyFunc := func(token *jwt.Token) (interface{}, error) {
				_, ok := token.Method.(*jwt.SigningMethodHMAC)
				if !ok {
					return nil, status.Errorf(codes.Unauthenticated, "unexpected signing method: %v", token.Header["alg"])
				}

				return []byte(a.conf.Auth.SecretKey), nil
			}

			accessToken, err := a.extractTokenFromHeader(req.Header())
			if err != nil {
				return nil, err
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

			a.logger.Info("claims", zap.Any("claims", claims))

			userID, found := claims["email"].(string)
			if !found {
				a.logger.Error("unable to get user id from token", zap.Any("claims", claims))

				return nil, status.Errorf(codes.Unauthenticated, "unable to get user id from token")
			}

			user, err := a.repo.GetUserFromEmail(ctx, userID)
			if err != nil {
				a.logger.Error("error authenticating user", zap.Error(err))

				return nil, status.Errorf(codes.Internal, "error authenticating user")
			}

			if user == nil {
				return nil, status.Errorf(codes.NotFound, "user not found")
			}

			ctx = context.WithValue(ctx, UserKey{}, user)

			return next(ctx, req)
		}
	}
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
