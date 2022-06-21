package hocoosmiddleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(&HocoosMiddleware{})
	httpcaddyfile.RegisterHandlerDirective("hocoos_middleware", parseCaddyfile)
}

type HocoosMiddleware struct {
	RedisURL     string `json:"redis_url"`
	PathPrefix   string `json:"path_prefix"`
	CacheTTL     int    `json:"cache_ttl"`
	ExcludeHosts string `json:"exclude_hosts"`

	logger       *zap.SugaredLogger
	client       *redisClient
	localCache   *localCache
	excludeHosts []string
}

// CaddyModule returns the Caddy module information.
func (m *HocoosMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.hocoos_middleware",
		New: func() caddy.Module { return new(HocoosMiddleware) },
	}
}

// Provision implements caddy.Provisioner.
func (m *HocoosMiddleware) Provision(ctx caddy.Context) error {
	if err := m.Validate(); err != nil {
		return err
	}
	m.logger = ctx.Logger(m).Sugar()
	if m.PathPrefix == "" {
		m.PathPrefix = "Caddy/Config/Domains"
	}

	if m.CacheTTL <= 0 {
		m.CacheTTL = 60
	}
	m.excludeHosts = strings.Split(m.ExcludeHosts, ",")
	m.localCache = newLocalCache(time.Duration(m.CacheTTL) * time.Second)
	m.client = newRedisClient()
	if err := m.client.init(ctx, m.RedisURL, m.logger); err != nil {
		return err
	}
	return nil
}

// Validate implements caddy.Validator.
func (m *HocoosMiddleware) Validate() error {
	if m.RedisURL == "" {
		return fmt.Errorf("redis_url is required")
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m *HocoosMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	m.logger.Debugf("%s %s", r.Method, r.URL.Path)
	for _, host := range m.excludeHosts {
		if strings.Contains(r.Host, host) {
			m.logger.Debugf("host %s excluded", r.Host)
			return next.ServeHTTP(w, r)
		}
	}
	key := fmt.Sprintf("%s/%s", m.PathPrefix, r.Host)
	var data string
	var found bool
	data, found = m.localCache.Get(key)
	if found {
		m.logger.Debugf("%s %s found in local cache", r.Method, r.URL.Path)
	} else {
		var err error
		data, err = m.client.get(r.Context(), key)
		if err != nil {
			m.logger.Debugf("get %s error: %v", key, err)
			w.WriteHeader(http.StatusInternalServerError)
			_, writeErr := w.Write([]byte(err.Error()))
			return writeErr
		}
		m.logger.Debugf("get %s: %s from redis", key, data)
		m.localCache.Set(key, data)
	}

	switch data {
	case "0":
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(fmt.Sprintf("%s host not allowed", r.Host)))
		return err
	case "1":
		return next.ServeHTTP(w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		_, writeErr := w.Write([]byte(fmt.Sprintf("permission data for host %s not found", r.Host)))
		return writeErr
	}
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *HocoosMiddleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		key := d.Val()
		var value string
		if !d.Args(&value) {
			continue
		}

		switch key {
		case "redis_url":
			m.RedisURL = value
		case "path_prefix":
			m.PathPrefix = value
		case "cache_ttl":
			itoa, err := strconv.Atoi(value)
			if err != nil {
				m.CacheTTL = 60
			} else {
				m.CacheTTL = itoa
			}
		case "exclude_hosts":
			m.ExcludeHosts = value

		default:
			return fmt.Errorf("unknown key %s", key)
		}
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new HocoosMiddleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m *HocoosMiddleware
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return m, err
}

// Interface guards
var (
	_ caddy.Provisioner           = (*HocoosMiddleware)(nil)
	_ caddy.Validator             = (*HocoosMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*HocoosMiddleware)(nil)
	_ caddyfile.Unmarshaler       = (*HocoosMiddleware)(nil)
)
