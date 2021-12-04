package rebinder

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

const name = "rebind"

var bindMap = make(map[string]*Node)

type Rebinder struct {
	Next       plugin.Handler
	CacheTimer time.Duration
	CacheLimit int
}

type Node struct {
	value net.IP
	count int
	next  *Node
}

func (rb Rebinder) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	// Only respond to A (1), CNAME (5)
	switch r.Question[0].Qtype {
	case 1, 5:
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

	// Check if query is in cache, else add to cache
	if val, ok := bindMap[label]; ok {
		answer = val.value
	} else {
		// if cache is full, ignore query
		if len(bindMap) >= rb.CacheLimit {
			return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// checks for malformed query
		if len(tokens) < 6 {
			return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 1: first IP address (hex encoded)
		fst, err := decodeHexIP(tokens[1])
		if err != nil {
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 2: first IP repeat pattern
		fstRepeat, err := strconv.Atoi(tokens[2])
		if err != nil || fstRepeat < 1 {
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 3: second IP address (hex encoded)
		snd, err := decodeHexIP(tokens[3])
		if err != nil {
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 4: second IP repeat pattern
		sndRepeat, err := strconv.Atoi(tokens[4])
		if err != nil || sndRepeat < 1 {
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// correctly formed query, add to cache
		next := Node{value: snd, count: sndRepeat}
		bindMap[label] = &Node{value: fst, count: fstRepeat, next: &next}

		// expire entry after specified duration
		time.AfterFunc(rb.CacheTimer, func() { delete(bindMap, label) })

		// answer query
		answer = fst
	}

	// respond
	a := new(dns.Msg)
	a.SetReply(r)
	a.Authoritative = true

	// only A records supported so far
	var rr dns.RR
	rr = new(dns.A)
	rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
	rr.(*dns.A).A = answer.To4()
	a.Answer = append(a.Answer, rr)

	// next node
	if bindMap[label].count > 1 {
		bindMap[label].count--
	} else if bindMap[label].next != nil {
		bindMap[label] = bindMap[label].next
	} else {
		delete(bindMap, label)
	}

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
