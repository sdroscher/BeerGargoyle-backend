package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	grpcreflect "github.com/bufbuild/connect-grpcreflect-go"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"droscher.com/BeerGargoyle/configs"
	"droscher.com/BeerGargoyle/pkg/auth"
	"droscher.com/BeerGargoyle/pkg/repository"
	"droscher.com/BeerGargoyle/pkg/server"
	"droscher.com/BeerGargoyle/pkg/server/grpc/api/v1/apiv1connect"
)

const timeout = 5 * time.Second

type ServeCmd struct {
	ConfigFile string `default:".BeerGargoyle.toml" help:"Path to config file" short:"c"`
}

func (s *ServeCmd) Run(_ *Context) error {
	logConfig := zap.NewProductionConfig()

	logger, _ := logConfig.Build()
	defer logger.Sync() //nolint:errcheck // we don't care about logger sync errors

	conf, err := configs.GetConfig(s.ConfigFile, logger)
	if err != nil {
		logger.Error("error loading config", zap.Error(err))

		return err
	}

	repo, err := repository.Open(conf, logger)
	if err != nil {
		logger.Error("error connecting to database", zap.Error(err))

		return err
	}
	defer repo.Close()

	authManager := auth.NewAuthManager(conf, repo, logger)
	interceptors := connect.WithInterceptors(authManager.GrpcAuthInterceptor())

	mux := http.NewServeMux()

	path, handler := apiv1connect.NewBeerServiceHandler(server.NewBeerServer(repo, logger, conf), interceptors)
	mux.Handle(path, handler)

	path, handler = apiv1connect.NewUserServiceHandler(server.NewUserServer(repo, logger), interceptors)
	mux.Handle(path, handler)

	path, handler = apiv1connect.NewCellarServiceHandler(server.NewCellarServer(repo, repo, repo, logger), interceptors)
	mux.Handle(path, handler)

	reflector := grpcreflect.NewStaticReflector(grpchealth.HealthV1ServiceName, apiv1connect.BeerServiceName, apiv1connect.UserServiceName, apiv1connect.CellarServiceName)
	checker := grpchealth.NewStaticChecker(apiv1connect.BeerServiceName, apiv1connect.UserServiceName, apiv1connect.CellarServiceName)
	mux.Handle(grpchealth.NewHandler(checker))
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	address := fmt.Sprintf(":%d", conf.Server.Port)

	// Configure CORS first
	corsHandler := configureCORS(mux)
	serverHandler := h2c.NewHandler(corsHandler, &http2.Server{})

	svr := &http.Server{
		Addr:              address,
		ReadHeaderTimeout: timeout,
		Handler:           serverHandler,
	}

	err = svr.ListenAndServe()
	if err != nil {
		logger.Error("failed to start server", zap.Error(err))

		return err
	}

	return nil
}

func configureCORS(mux *http.ServeMux) http.Handler {
	corsOpts := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"},
		AllowedHeaders: []string{
			"accept",
			"accept-encoding",
			"accept-language",
			"authorization",
			"cache-control",
			"connect-accept-encoding",
			"connect-content-encoding",
			"connect-protocol-version",
			"connect-timeout-ms",
			"content-encoding",
			"content-length",
			"content-type",
			"custom-header-1",
			"date",
			"grpc-accept-encoding",
			"grpc-encoding",
			"grpc-message",
			"grpc-status",
			"grpc-status-details-bin",
			"grpc-timeout",
			"keep-alive",
			"origin",
			"referer",
			"user-agent",
			"x-accept-content-transfer-encoding",
			"x-accept-response-streaming",
			"x-grpc-web",
			"x-user-agent",
		},
		ExposedHeaders: []string{
			"connect-protocol-version",
			"grpc-message",
			"grpc-status",
			"grpc-status-details-bin",
		},
		MaxAge:             86400, // 24 hours
		OptionsPassthrough: false, // Handle OPTIONS requests in CORS middleware
	})

	// Apply CORS to the main mux, then wrap with h2c
	corsHandler := corsOpts.Handler(mux)

	return corsHandler
}
