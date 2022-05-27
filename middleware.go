package hocoosmiddleware

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(HocoosMiddleware{})
	httpcaddyfile.RegisterHandlerDirective("hocoos_middleware", parseCaddyfile)
}

// HocoosMiddleware implements an HTTP handler that writes the
// visitor's IP address to a file or stream.
type HocoosMiddleware struct {
	// The file or stream to write to. Can be "stdout"
	// or "stderr".
	Output string `json:"output,omitempty"`

	w io.Writer
}

// CaddyModule returns the Caddy module information.
func (HocoosMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.hocoos_middleware",
		New: func() caddy.Module { return new(HocoosMiddleware) },
	}
}

// Provision implements caddy.Provisioner.
func (m *HocoosMiddleware) Provision(ctx caddy.Context) error {
	switch m.Output {
	case "stdout":
		m.w = os.Stdout
	case "stderr":
		m.w = os.Stderr
	default:
		return fmt.Errorf("an output stream is required")
	}
	return nil
}

// Validate implements caddy.Validator.
func (m *HocoosMiddleware) Validate() error {
	if m.w == nil {
		return fmt.Errorf("no writer")
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m HocoosMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	m.w.Write([]byte(r.RemoteAddr))
	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *HocoosMiddleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&m.Output) {
			return d.ArgErr()
		}
	}
	return nil
}

// parseCaddyfile unmarshals tokens from h into a new HocoosMiddleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m HocoosMiddleware
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
