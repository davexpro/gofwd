package main

import (
	"net"
	"time"

	"github.com/bytedance/gopkg/cache/asynccache"
	"github.com/bytedance/gopkg/lang/fastrand"
	"github.com/miekg/dns"
)

var (
	dnsCli   *dns.Client
	dnsCache asynccache.AsyncCache

	dotSrvs = []string{"1.1.1.1:853", "8.8.8.8:853"}
)

const (
	defaultAnswer = "127.0.0.1"
)

func init() {
	dnsCache = asynccache.NewAsyncCache(asynccache.Options{
		RefreshDuration: time.Minute * 3,
		Fetcher: func(domain string) (interface{}, error) {
			return resolveDomainIP(domain), nil
		},
		EnableExpire: false,
	})

	dnsCli = &dns.Client{Net: "tcp-tls", Timeout: 5 * time.Second}
}

func resolveDomainIP(domain string) string {
	if net.ParseIP(domain) != nil {
		return domain
	}

	req := new(dns.Msg)
	req.SetQuestion(domain+".", dns.TypeA)
	resp, _, err := dnsCli.Exchange(req, dotSrvs[fastrand.Int()%len(dotSrvs)])
	if err != nil {
		return defaultAnswer
	}

	for _, answer := range resp.Answer {
		switch answer.(type) {
		case *dns.A:
			return answer.(*dns.A).A.String()
		case *dns.AAAA:
			//	res = append(res, answer.(*dns.AAAA).AAAA.String())
		}
	}

	return defaultAnswer
}

func resolveUDPAddress(address string) (*net.UDPAddr, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	// if it's an IP address, return directly
	if net.ParseIP(host) != nil {
		return net.ResolveUDPAddr("udp", address)
	}

	// if it's a domain, resolve it
	ip, err := dnsCache.Get(host)
	if err != nil {
		return nil, err
	}

	return net.ResolveUDPAddr("udp", net.JoinHostPort(ip.(string), portStr))
}
