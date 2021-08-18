package rebinder

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

const name = "rebind"

var bindMap = make(map[string]*Node)

type Rebinder struct{ Next plugin.Handler }

type Node struct {
	value net.IP
	next  *Node
}

func (rb Rebinder) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	// Only respond to A (1), CNAME (5) and AAAA (28) requests
	switch r.Question[0].Qtype {
	case 1, 5, 28:
		{
			break
		}
	default:
		{
			return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}
	}

	// Parse query
	tokens := strings.Split(state.Name(), ".")
	label := tokens[0]
	var answer net.IP

	// check if query is in cache
	if val, ok := bindMap[label]; ok {
		answer = val.value
	} else {
		// add query to cache
		if len(tokens) < 4 {
			return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		fst, err := decodeHexIP(tokens[1])
		if err != nil {
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		snd, err := decodeHexIP(tokens[2])
		if err != nil {
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		next := Node{value: snd}
		bindMap[label] = &Node{value: fst, next: &next}
		next.next = bindMap[label]

		answer = fst
	}

	// Respond
	a := new(dns.Msg)
	a.SetReply(r)
	a.Authoritative = true

	var rr dns.RR
	/*
		switch state.Family() {
		case 1:
			rr = new(dns.A)
			rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
			rr.(*dns.A).A = answer.To4()
		case 2:
			rr = new(dns.A)
			rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass()}
			rr.(*dns.AAAA).AAAA = answer
		}
	*/
	rr = new(dns.A)
	rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
	rr.(*dns.A).A = answer.To4()

	a.Answer = append(a.Answer, rr)

	// Next node
	bindMap[label] = bindMap[label].next

	w.WriteMsg(a)
	return 0, nil
}

func decodeHexIP(hexIp string) (net.IP, error) {
	// hex to string
	ipStr, err := hex.DecodeString(hexIp)
	if err != nil {
		return nil, fmt.Errorf("Invalid Hex IP: %s", hexIp)
	}
	// string to IP
	ip := net.ParseIP(string(ipStr))
	if ip == nil {
		return nil, fmt.Errorf("Invalid IP: %s", ipStr)
	}

	return ip, nil
}

func (wh Rebinder) Name() string { return name }
