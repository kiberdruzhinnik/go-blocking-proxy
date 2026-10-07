package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	socksVersion = 5
	methodNone   = 0
	methodReject = 0xff
	cmdConnect   = 1
	replyBlocked = 2 // connection not allowed by ruleset
	replyCommand = 7 // command not supported
)

func main() {
	listen := flag.String("listen", "127.0.0.1:1080", "SOCKS5 listen address (use a LAN IP or :1080 for router access)")
	flag.Parse()

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen on %s: %v", *listen, err)
	}
	log.Printf("blocking SOCKS5 proxy listening on %s", listener.Addr())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if errors.Is(err, syscall.EINTR) || errors.Is(err, syscall.ECONNABORTED) {
				log.Printf("accept: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.Printf("accept: %v", err)
			return
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	var greeting [2]byte
	if _, err := io.ReadFull(conn, greeting[:]); err != nil || greeting[0] != socksVersion || greeting[1] == 0 {
		return
	}
	methods := make([]byte, int(greeting[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return
	}
	for _, method := range methods {
		if method == methodNone {
			if _, err := conn.Write([]byte{socksVersion, methodNone}); err != nil {
				return
			}
			serveRequest(conn)
			return
		}
	}
	_, _ = conn.Write([]byte{socksVersion, methodReject})
}

func serveRequest(conn net.Conn) {
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil || header[0] != socksVersion || header[2] != 0 {
		return
	}

	var host string
	switch header[3] {
	case 1: // IPv4
		addr := make([]byte, net.IPv4len)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return
		}
		host = net.IP(addr).String()
	case 3: // Domain name
		var size [1]byte
		if _, err := io.ReadFull(conn, size[:]); err != nil || size[0] == 0 {
			return
		}
		name := make([]byte, int(size[0]))
		if _, err := io.ReadFull(conn, name); err != nil {
			return
		}
		host = string(name)
	case 4: // IPv6
		addr := make([]byte, net.IPv6len)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return
		}
		host = net.IP(addr).String()
	default:
		return
	}

	var portBytes [2]byte
	if _, err := io.ReadFull(conn, portBytes[:]); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBytes[:])

	reply := byte(replyBlocked)
	if header[1] != cmdConnect {
		reply = replyCommand
	}
	if err := writeReply(conn, reply); err != nil {
		return
	}
	log.Printf("blocked SOCKS5 request %s:%d from %s (command %d)", strings.TrimSpace(host), port, conn.RemoteAddr(), header[1])
}

func writeReply(conn net.Conn, code byte) error {
	// A successful SOCKS5 response would contain a bound address. The zero
	// address here is sufficient for an error response.
	_, err := conn.Write([]byte{socksVersion, code, 0, 1, 0, 0, 0, 0, 0, 0})
	if err != nil {
		return fmt.Errorf("write SOCKS5 reply: %w", err)
	}
	return nil
}
