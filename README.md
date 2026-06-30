[![Go](https://github.com/sdroscher/BeerGargoyle-backend/actions/workflows/go.yml/badge.svg)](https://github.com/sdroscher/BeerGargoyle-backend/actions/workflows/go.yml)

A beer cellar management backend API by Simon Droscher

## Beer search (Untappd)

Beer and brewery search are served by Untappd's **Algolia** search backend. The
server queries Algolia's JSON API directly (`FindBeer` / `FindBrewery`), which:

- returns structured data (name, brewery, ABV, IBU, style, rating, image), and
- is **not** behind Cloudflare, so it works from any host (including Fly.io)
  with no proxy.

The public search-only Algolia credentials are embedded in Untappd's web pages
and ship as defaults. If Untappd ever rotates them, override via config or env
vars (no redeploy of code needed) — see `[Integrations.Untappd]` in
`.BeerGargoyle.toml.example`:

```bash
fly secrets set BEERGARGOYLE_INTEGRATIONS_UNTAPPD_ALGOLIAAPPID=...
fly secrets set BEERGARGOYLE_INTEGRATIONS_UNTAPPD_ALGOLIASEARCHKEY=...
```

## Beer descriptions — Cloudflare proxy setup

Algolia search does **not** include the long beer/brewery descriptions — those
live only on Untappd's individual beer/brewery pages, and the entire
`untappd.com` domain is behind Cloudflare. Cloudflare serves a managed challenge
(`cf-mitigated: challenge`, "Just a moment…") to **datacenter IPs** (such as
Fly.io), so those pages can't be fetched directly from production. Residential
IPs are not challenged, so this works without a proxy in local development.

Descriptions are therefore fetched lazily and only **when a beer is saved**
(server-side, in `AddBeer`) — one request per saved beer, not per search result.
To make that work in production, route those detail-page fetches through a proxy
with a clean (residential) IP, configured under `[Integrations.Untappd]`.

If no proxy is configured, search and saving still work; the beer is just saved
without a description when running from a blocked IP.

### Option A — Scrape.do free tier (recommended)

[Scrape.do](https://scrape.do) has a no-credit-card free tier of **1,000
successful calls/month** including residential proxies and Cloudflare bypass —
far more than save-time description fetches will ever use. It's used via its
proxy-mode endpoint, so it plugs straight into the generic `Proxy` setting:

```toml
[Integrations.Untappd]
Proxy = "http://YOUR_TOKEN:super=true&render=false@proxy.scrape.do:8080"
ProxyInsecureSkipVerify = true
```

Or on Fly.io:

```bash
fly secrets set BEERGARGOYLE_INTEGRATIONS_UNTAPPD_PROXY='http://YOUR_TOKEN:super=true&render=false@proxy.scrape.do:8080'
fly secrets set BEERGARGOYLE_INTEGRATIONS_UNTAPPD_PROXYINSECURESKIPVERIFY=true
```

- `super=true` → residential IP (untappd doesn't challenge residential IPs).
- `render=false` → no JS rendering needed; the description is in the page's
  server-rendered `ld+json`. If a request ever returns a challenge page, switch
  to `render=true`.

Other providers with free/cheap tiers work the same way via their proxy-mode
URLs, e.g. ScrapingAnt (`proxy.scrapingant.com`) or pay-as-you-go residential
proxies (IPRoyal, Bright Data) — cheap here because each page is ~150 KB.

### Option B — Generic proxy

Any HTTP proxy with a residential/ISP IP works (including a self-hosted proxy on
a home connection):

```toml
[Integrations.Untappd]
Proxy = "http://user:pass@host:port"
# Set true only if the proxy performs TLS interception:
ProxyInsecureSkipVerify = false
```

### ScraperAPI

There is also a dedicated `ScraperAPIKey` setting (used only when `Proxy` is
empty), but note ScraperAPI no longer offers a free tier — prefer Option A.

`Proxy` takes precedence over `ScraperAPIKey`. Leave both unset to fetch
descriptions directly (fine for local development from a residential connection).
