package rebinder

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

const name = "rebinder"

type Rebinder struct {
	Next plugin.Handler
}

func (rb Rebinder) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return 0, nil
}

func (wh Rebinder) Name() string { return name }
