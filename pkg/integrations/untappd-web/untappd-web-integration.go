package untappdweb

import "go.uber.org/zap"

const IntegrationName = "untappd_web"

type UntappedWebIntegration struct {
	logger *zap.Logger
}

func NewUntappedWebIntegration(logger *zap.Logger) *UntappedWebIntegration {
	return &UntappedWebIntegration{logger: logger}
}
