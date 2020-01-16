package ehco

import (
	"log"
	"net"
	"time"
)

type Relay struct {
	LocalTCPAddr  *net.TCPAddr
	LocalUDPAddr  *net.UDPAddr
	RemoteTCPAddr *net.TCPAddr
	RemoteUDPAddr *net.UDPAddr
	TCPListener   *net.TCPListener
	UDPConn       *net.UDPConn

	TCPDeadline int
	UDPDeadline int
}

func NewRelay(localAddr, remoteAddr string, tcpTimeout, tcpDeadline, udpDeadline int) (*Relay, error) {
	localTCPAddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		return nil, err
	}
	localUDPAddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	remoteTCPAddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
	if err != nil {
		return nil, err
	}
	remoteUDPAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return nil, err
	}

	s := &Relay{
		LocalTCPAddr:  localTCPAddr,
		LocalUDPAddr:  localUDPAddr,
		RemoteTCPAddr: remoteTCPAddr,
		RemoteUDPAddr: remoteUDPAddr,

		TCPTimeout:  tcpTimeout,
		TCPDeadline: tcpDeadline,
		UDPDeadline: udpDeadline,
	}
	return s, nil
}

func (relay *Relay) ListenAndServe() error {
	errChan := make(chan error)
	go func() {
		errChan <- relay.RunLocalTCPServer()
	}()
	go func() {
		errChan <- relay.RunLocalUDPServer()
	}()
	return <-errChan
}

func (relay *Relay) RunLocalTCPServer() error {
	var err error
	relay.TCPListener, err = net.ListenTCP("tcp", relay.LocalTCPAddr)
	if err != nil {
		return err
	}
	defer relay.TCPListener.Close()
	for {
		c, err := relay.TCPListener.AcceptTCP()
		if err != nil {
			return err
		}
		go func(c *net.TCPConn) {
			defer c.Close()
			if relay.TCPDeadline != 0 {
				if err := c.SetDeadline(time.Now().Add(time.Duration(relay.TCPDeadline) * time.Second)); err != nil {
					log.Println(err)
					return
				}
			}
			if err := relay.HandleTCPConn(c); err != nil {
				log.Println(err)
			}
		}(c)
	}
	return nil
}

func (relay *Relay) RunLocalUDPServer() error {
	var err error
	relay.UDPConn, err = net.ListenUDP("udp", relay.LocalUDPAddr)
	if err != nil {
		return err
	}
	defer relay.UDPConn.Close()
	for {
		b := make([]byte, 65536)
		n, addr, err := relay.UDPConn.ReadFromUDP(b)
		if err != nil {
			return err
		}
		go func(addr *net.UDPAddr, b []byte) {
			if err := relay.HandleUDP(addr, b); err != nil {
				log.Println(err)
				return
			}
		}(addr, b[0:n])
	}
	return nil
}

func (relay *Relay) Shutdown() error {
	var err, err1 error
	if relay.TCPListener != nil {
		err = relay.TCPListener.Close()
	}
	if relay.UDPConn != nil {
		err1 = relay.UDPConn.Close()
	}
	if err != nil {
		return err
	}
	return err1
}

func (relay *Relay) HandleTCPConn(c *net.TCPConn) error {
	rc, err := net.Dial("tcp", relay.RemoteTCPAddr.String())
	if err != nil {
		return err
	}
	defer rc.Close()

	if relay.TCPDeadline != 0 {
		if err := rc.SetDeadline(time.Now().Add(time.Duration(relay.TCPDeadline) * time.Second)); err != nil {
			return err
		}
	}

	go func() {
		var bf [1024 * 2]byte
		for {
			if relay.TCPDeadline != 0 {
				if err := rc.SetDeadline(time.Now().Add(time.Duration(relay.TCPDeadline) * time.Second)); err != nil {
					return
				}
			}
			i, err := rc.Read(bf[:])
			if err != nil {
				return
			}
			if _, err := c.Write(bf[0:i]); err != nil {
				return
			}
		}
	}()

	var bf [1024 * 2]byte
	for {
		if relay.TCPDeadline != 0 {
			if err := c.SetDeadline(time.Now().Add(time.Duration(relay.TCPDeadline) * time.Second)); err != nil {
				return nil
			}
		}
		i, err := c.Read(bf[:])
		if err != nil {
			return nil
		}
		if _, err := rc.Write(bf[0:i]); err != nil {
			return nil
		}
	}
	return nil
}

func (relay *Relay) HandleUDP(addr *net.UDPAddr, b []byte) error {
	return nil
}
