package rebinder

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register("rebinder", rebinder) }

func rebinder(c *caddy.Controller) error {
	c.Next()

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Rebinder{}
	})

	return nil
}
