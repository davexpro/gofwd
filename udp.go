package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	udpFlags = []cli.Flag{
		&cli.StringFlag{
			Name:    "listen",
			Aliases: []string{"l"},
			Value:   "[::]:4096",
		},
		&cli.StringFlag{
			Name:    "target",
			Aliases: []string{"t"},
			Value:   "1.1.1.1:53",
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
			Value:   false,
		},
	}
)

type UDPForward struct {
	from, to string
	fromAddr *net.UDPAddr
	listener *net.UDPConn
	timeout  time.Duration

	closed   bool
	verbose  bool
	prepared bool
}

func NewUDPFwd(from, to string, timeout time.Duration, verbose bool) *UDPForward {
	return &UDPForward{
		from:    from,
		to:      to,
		timeout: timeout,
		verbose: verbose,
	}
}

func (f *UDPForward) prepare() error {
	var err error
	f.fromAddr, err = net.ResolveUDPAddr("udp", f.from)
	if err != nil {
		return err
	}

	_, err = net.ResolveUDPAddr("udp", f.to)
	if err != nil {
		return err
	}

	f.prepared = true
	return nil
}

func (f *UDPForward) run() error {
	if !f.prepared {
		log.Println("UDPForward not prepared, call .prepare() first")
		return nil
	}

	var err error
	f.listener, err = net.ListenUDP("udp", f.fromAddr)
	if err != nil {
		return err
	}

	go f.forward()
	go f.recycle()

	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := f.shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 3 seconds.")
	}
	log.Println("Server exiting")

	return nil
}

func (f *UDPForward) shutdown(ctx context.Context) error {
	f.closed = true
	return nil
}

func (f *UDPForward) forward() {
	for {
		buf, oob := make([]byte, bufferSize), make([]byte, bufferSize)
		n, _, _, addr, err := f.listener.ReadMsgUDP(buf, oob)
		if err != nil {
			log.Println("forward: failed to read, terminating:", err)
			return
		}
		go f.handleConn(buf[:n], addr)
	}
}

func (f *UDPForward) recycle() {
	for !f.closed {
		time.Sleep(f.timeout)
		delKeys := make([]string, 0, 32)
		bounder := time.Now().Add(-f.timeout).Unix()
		connHub.Range(func(key, value interface{}) bool {
			if value.(*conn).accessTime < bounder {
				delKeys = append(delKeys, key.(string))
			}
			return true
		})
		for _, key := range delKeys {
			connHub.Delete(key)
		}
		log.Println("recycle: cleaned up", len(delKeys), "connections")
	}
}

func (f *UDPForward) handleConn(data []byte, fromAddr *net.UDPAddr) {
	raw, loaded := connHub.LoadOrStore(fromAddr.String(), &conn{accessTime: time.Now().Unix()})
	c := raw.(*conn)

	if !loaded {
		log.Println("recv conn from:", fromAddr.String(), "to", f.to)
	}

	toAddr, err := resolveUDPAddress(f.to)
	if err != nil {
		log.Println("gofwd: failed to resolve:", err)
		connHub.Delete(fromAddr.String())
		return
	}

	if c.udp != nil && toAddr != nil {
		if c.udp.RemoteAddr().String() != toAddr.String() {
			log.Println("gofwd: remote fromAddr changed, closing:", c.udp.RemoteAddr().String(), toAddr.String())
			c.udp.Close()
			c.udp = nil
		}
	}

	if c.udp == nil {
		udpConn, err := net.DialUDP("udp", nil, toAddr)
		if err != nil {
			log.Println("gofwd: failed to dial:", err)
			connHub.Delete(fromAddr.String())
			return
		}

		c.udp = udpConn
		c.accessTime = time.Now().Unix()
		_, _, err = udpConn.WriteMsgUDP(data, nil, nil)
		if err != nil {
			log.Println("gofwd: error sending initial packet to client", err)
		}

		for {
			// log.Println("in loop to read from NAT conn to servers")
			buf, oob := make([]byte, bufferSize), make([]byte, bufferSize)
			n, _, _, _, err := udpConn.ReadMsgUDP(buf, oob)
			if err != nil {
				udpConn.Close()
				connHub.Delete(fromAddr.String())
				if !errors.Is(err, net.ErrClosed) {
					log.Println("gofwd: abnormal read, closing:", err)
				}
				return
			}

			if f.verbose {
				log.Println("->", "size", n, "from", fromAddr.String(), "to", udpConn.RemoteAddr())
			}
			_, _, err = f.listener.WriteMsgUDP(buf[:n], nil, fromAddr)
			if err != nil {
				log.Println("gofwd: error sending packet to client:", err)
			}
		}
	}

	if f.verbose {
		log.Println("<-", "size", len(data), "from", c.udp.RemoteAddr(), "to", fromAddr.String())
	}
	_, _, err = c.udp.WriteMsgUDP(data, nil, nil)
	if err != nil {
		log.Println("gofwd: error sending packet to server:", err)
	}

	c.accessTime = time.Now().Unix()
}
