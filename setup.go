package rebinder

import (
	"fmt"
	"strconv"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/log"
)

func init() {
	plugin.Register("rebind", setup)
	log.Debug("loaded rebind plugin")
}

func setup(c *caddy.Controller) error {
	c.Next() // rebinder
	c.NextBlock()

	var cacheTimer time.Duration = 300
	var cacheLimit int = 10000
	var err error

	switch c.Val() {
	case "cacheTimer":
		if !c.NextArg() {
			return fmt.Errorf("Missing value for cacheTimer")
		}
		cacheTimer, err = time.ParseDuration(c.Val())
		if err != nil {
			return fmt.Errorf("Invalid value for cacheTimer")
		}
	case "cacheLimit":
		if !c.NextArg() {
			return fmt.Errorf("Missing value for cacheLimit")
		}
		cacheLimit, err = strconv.Atoi(c.Val())
		if err != nil {
			return fmt.Errorf("Invalid value for cacheLimit")
		}
	default:
		return fmt.Errorf(("Unknown argument"))
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Rebinder{
			Next:       next,
			CacheTimer: cacheTimer,
			CacheLimit: cacheLimit,
		}
	})

	return nil
}
