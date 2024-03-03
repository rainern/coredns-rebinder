package rebinder

import (
	"fmt"
	"strconv"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register("rebind", setup)
}

func setup(c *caddy.Controller) error {
	c.Next() // rebinder

	var cacheTimer time.Duration = 300
	var cacheLimit int = 10000
	var ttl int = 1
	var err error

	for c.NextBlock() {
		switch c.Val() {
		case "ttl":
			if !c.NextArg() {
				return fmt.Errorf("missing value for ttl")
			}
			ttl, err = strconv.Atoi(c.Val())
			if err != nil {
				return fmt.Errorf("invalid value for ttl")
			}
		case "cacheTimer":
			if !c.NextArg() {
				return fmt.Errorf("missing value for cacheTimer")
			}
			cacheTimer, err = time.ParseDuration(c.Val())
			if err != nil {
				return fmt.Errorf("invalid value for cacheTimer")
			}
		case "cacheLimit":
			if !c.NextArg() {
				return fmt.Errorf("missing value for cacheLimit")
			}
			cacheLimit, err = strconv.Atoi(c.Val())
			if err != nil {
				return fmt.Errorf("invalid value for cacheLimit")
			}
		default:
			return fmt.Errorf("unknown argument: %s", c.Val())
		}
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Rebinder{
			Next:       next,
			CacheTimer: cacheTimer,
			CacheLimit: cacheLimit,
			Ttl:        ttl,
		}
	})

	return nil
}
