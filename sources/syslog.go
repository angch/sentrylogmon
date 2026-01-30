package sources

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type SyslogSource struct {
	name      string
	network   string
	address   string
	listener  io.Closer
	reader    *io.PipeReader
	writer    *io.PipeWriter
	wg        sync.WaitGroup
	closeChan chan struct{}
}

func NewSyslogSource(name string, address string) *SyslogSource {
	// Parse network from address if present (e.g. "tcp:0.0.0.0:514")
	network := "udp"
	addr := address
	if strings.Contains(address, ":") {
		// Detect if it starts with tcp: or udp:
		if strings.HasPrefix(address, "tcp:") {
			network = "tcp"
			addr = strings.TrimPrefix(address, "tcp:")
		} else if strings.HasPrefix(address, "udp:") {
			network = "udp"
			addr = strings.TrimPrefix(address, "udp:")
		}
	}

	return &SyslogSource{
		name:      name,
		network:   network,
		address:   addr,
		closeChan: make(chan struct{}),
	}
}

func (s *SyslogSource) Name() string {
	return s.name
}

func (s *SyslogSource) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	if l, ok := s.listener.(net.Listener); ok {
		return l.Addr()
	}
	if c, ok := s.listener.(net.PacketConn); ok {
		return c.LocalAddr()
	}
	return nil
}

func (s *SyslogSource) Close() error {
	select {
	case <-s.closeChan:
		return nil
	default:
		close(s.closeChan)
	}

	if s.listener != nil {
		s.listener.Close()
	}

	// We don't close writer here immediately, we let the goroutine do it when listener closes/fails
	// to ensure we drain or finish properly?
	// Actually, if we close listener, the loop exits, then it defers pw.Close().
	// So we just wait.

	s.wg.Wait()
	return nil
}

func (s *SyslogSource) Stream() (io.Reader, error) {
	pr, pw := io.Pipe()
	s.reader = pr
	s.writer = pw

	var err error
	if s.network == "tcp" {
		err = s.startTCP(pw)
	} else {
		err = s.startUDP(pw)
	}

	if err != nil {
		pw.Close()
		return nil, err
	}

	return pr, nil
}

func (s *SyslogSource) startUDP(pw *io.PipeWriter) error {
	addr, err := net.ResolveUDPAddr("udp", s.address)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %v", s.address, err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %v", s.address, err)
	}
	s.listener = conn

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer pw.Close()

		buf := make([]byte, 65536) // Max UDP size
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				// Check if this error is due to Close()
				select {
				case <-s.closeChan:
					return
				default:
					if !strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("Error reading from UDP syslog: %v", err)
					}
					return
				}
			}

			if n > 0 {
				// Append newline if missing to ensure scanner picks it up
				data := buf[:n]
				if data[n-1] != '\n' {
					// We need to append \n.
					out := make([]byte, n+1)
					copy(out, data)
					out[n] = '\n'
					if _, err := pw.Write(out); err != nil {
						return // Pipe closed
					}
				} else {
					if _, err := pw.Write(data); err != nil {
						return // Pipe closed
					}
				}
			}
		}
	}()
	return nil
}

func (s *SyslogSource) startTCP(pw *io.PipeWriter) error {
	addr, err := net.ResolveTCPAddr("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to resolve TCP address %s: %v", s.address, err)
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on TCP %s: %v", s.address, err)
	}
	s.listener = ln

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer pw.Close()

		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				select {
				case <-s.closeChan:
					return
				default:
					if !strings.Contains(err.Error(), "use of closed network connection") {
						log.Printf("Error accepting TCP connection: %v", err)
					}
					return
				}
			}

			s.wg.Add(1)
			go func(c *net.TCPConn) {
				defer s.wg.Done()
				defer c.Close()

				scanner := bufio.NewScanner(c)
				for scanner.Scan() {
					line := scanner.Bytes()
					// Write line + \n
					// To ensure atomicity in pipe (so lines don't get interleaved), write once.
					out := make([]byte, len(line)+1)
					copy(out, line)
					out[len(line)] = '\n'

					if _, err := pw.Write(out); err != nil {
						return // Pipe closed
					}
				}
			}(conn)
		}
	}()
	return nil
}
