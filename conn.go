package main

import (
	"net"
	"sync"
	"sync/atomic"
)

var (
	connHub sync.Map
)

type conn struct {
	lock       sync.RWMutex
	udp        *net.UDPConn
	accessTime int64
	sendCnt    atomic.Uint64
	recvCnt    atomic.Uint64
}
