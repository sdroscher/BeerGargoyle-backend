package untappdweb

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
)

const IntegrationName = "untappd_web"

// ProxyConfig configures an optional outbound proxy for detail-page requests,
// used to bypass Cloudflare challenges served to datacenter IPs.
type ProxyConfig struct {
	URL                string
	InsecureSkipVerify bool
}

// Config bundles everything the integration needs: Algolia search (no proxy
// required) and an optional proxy for Cloudflare-gated detail pages.
type Config struct {
	Proxy   ProxyConfig
	Algolia AlgoliaConfig
}

type UntappedWebIntegration struct {
	logger  *zap.Logger
	proxy   ProxyConfig
	algolia *algoliaClient
}

func NewUntappedWebIntegration(logger *zap.Logger, config Config) *UntappedWebIntegration {
	return &UntappedWebIntegration{
		logger:  logger,
		proxy:   config.Proxy,
		algolia: newAlgoliaClient(config.Algolia),
	}
}

// newCollector builds a collector with browser headers and the configured proxy
// applied. The proxy is set on this parent collector before any clones are made
// so that clones (which share colly's backend) inherit the transport.
func (u *UntappedWebIntegration) newCollector() *colly.Collector {
	collector := colly.NewCollector(
		colly.AllowedDomains("untappd.com"),
		colly.UserAgent(userAgent),
	)

	setBrowserHeaders(collector)
	u.applyProxy(collector)

	return collector
}

// applyProxy installs the configured proxy transport on the collector. It must
// be called on the parent collector before any clones or requests are made:
// colly's Clone() shares the underlying backend, so clones inherit the
// transport and setting it later from concurrent goroutines would race.
func (u *UntappedWebIntegration) applyProxy(collector *colly.Collector) {
	if u.proxy.URL == "" {
		return
	}

	proxyURL, err := url.Parse(u.proxy.URL)
	if err != nil {
		u.logger.Error("invalid untappd proxy URL; scraping without proxy", zap.Error(err))

		return
	}

	transport := &http.Transport{
		Proxy:             http.ProxyURL(proxyURL),
		DisableKeepAlives: true,
	}

	if u.proxy.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // proxy (e.g. ScraperAPI) performs TLS interception
	}

	collector.WithTransport(transport)

	u.logger.Info("untappd scraping via proxy", zap.String("proxy_host", proxyURL.Host))
}
