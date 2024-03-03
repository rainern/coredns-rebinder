package rebinder

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

const name = "rebind"

var log = clog.NewWithPlugin(name)

var bindMap = make(map[string]*Node)

type Rebinder struct {
	Next       plugin.Handler
	CacheTimer time.Duration
	CacheLimit int
	Ttl        int
}

type Node struct {
	value net.IP
	count int
	next  *Node
}

func (rb Rebinder) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	log.Debug("got rebind request: " + state.Name())

	// Only respond to A (1)
	if r.Question[0].Qtype != 1 {
		return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
	}

	// Parse query
	query := strings.SplitN(state.Name(), ".", 2)[0]
	tokens := strings.Split(query, "-")

	label := tokens[0]
	var answer net.IP

	// Check if query is in cache, else add to cache
	if val, ok := bindMap[label]; ok {
		answer = val.value
	} else {
		// if cache is full, ignore query
		if len(bindMap) >= rb.CacheLimit {
			log.Debug("cache full")
			return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// checks for malformed query
		if len(tokens) < 5 {
			log.Debug("malformed query")
			return plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 1: first IP address
		fst, err := encodeIp(tokens[1])
		if err != nil {
			log.Debug("first ip malformed")
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 2: first IP repeat pattern
		fstRepeat, err := strconv.Atoi(tokens[2])
		if err != nil || fstRepeat < 1 {
			log.Debug("first repeat pattern malformed")
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 3: second IP address
		snd, err := encodeIp(tokens[3])
		if err != nil {
			log.Debug("second ip malformed")
			plugin.NextOrFailure(state.Name(), rb.Next, ctx, w, r)
		}

		// token 4: second IP repeat pattern
		sndRepeat, err := strconv.Atoi(tokens[4])
		if err != nil || sndRepeat < 1 {
			log.Debug("second repeat pattern malformed")
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
	rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: uint32(rb.Ttl)}
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

func encodeIp(ipStr string) (net.IP, error) {
	// string to int
	ipInt, err := strconv.ParseUint(ipStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid IP: %s", ipStr)
	}

	// int to ip
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, uint32(ipInt))

	return ip, nil
}

func (wh Rebinder) Name() string { return name }
