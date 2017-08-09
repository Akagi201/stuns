package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/ernado/stun"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Server is RFC 5389 basic server implementation.
//
// Current implementation is UDP only and not utilizes FINGERPRINT mechanism,
// nor ALTERNATE-SERVER, nor credentials mechanisms. It does not support
// backwards compatibility with RFC 3489.
//
// The STUN server MUST support the Binding method.  It SHOULD NOT
// utilize the short-term or long-term credential mechanism.  This is
// because the work involved in authenticating the request is more than
// the work in simply processing it.  It SHOULD NOT utilize the
// ALTERNATE-SERVER mechanism for the same reason.  It MUST support UDP
// and TCP.  It MAY support STUN over TCP/TLS; however, TLS provides
// minimal security benefits in this basic mode of operation.  It MAY
// utilize the FINGERPRINT mechanism but MUST NOT require it.  Since the
// stand-alone server only runs STUN, FINGERPRINT provides no benefit.
// Requiring it would break compatibility with RFC 3489, and such
// compatibility is desirable in a stand-alone server.  Stand-alone STUN
// servers SHOULD support backwards compatibility with [RFC3489]
// clients, as described in Section 12.
//
// It is RECOMMENDED that administrators of STUN servers provide DNS
// entries for those servers as described in Section 9.
//
// A basic STUN server is not a solution for NAT traversal by itself.
// However, it can be utilized as part of a solution through STUN
// usages.  This is discussed further in Section 14.
type Server struct {
}

var (
	software = stun.NewSoftware("stuns")
)

var (
	errNotSTUNMessage = errors.New("not stun message")
)

func basicProcess(addr net.Addr, b []byte, req, res *stun.Message) error {
	if !stun.IsMessage(b) {
		return errNotSTUNMessage
	}
	if _, err := req.Write(b); err != nil {
		return errors.Wrap(err, "failed to read message")
	}
	var (
		ip   net.IP
		port int
	)
	switch a := addr.(type) {
	case *net.UDPAddr:
		ip = a.IP
		port = a.Port
	default:
		log.Fatalf("Unknown addr: %v", addr)
	}
	return res.Build(
		stun.NewTransactionIDSetter(req.TransactionID),
		stun.NewType(stun.MethodBinding, stun.ClassSuccessResponse),
		software,
		&stun.XORMappedAddress{
			IP:   ip,
			Port: port,
		},
		stun.Fingerprint,
	)
}

func (s *Server) serveConn(c net.PacketConn, res, req *stun.Message) error {
	if c == nil {
		return nil
	}
	buf := make([]byte, 1024)
	n, addr, err := c.ReadFrom(buf)
	if err != nil {
		log.Errorf("ReadFrom: %v", err)
		return nil
	}

	log.Debugf("read %d bytes from %s", n, addr)

	if _, err = req.Write(buf[:n]); err != nil {
		log.Errorf("Write: %v", err)
		return err
	}
	if err = basicProcess(addr, buf[:n], req, res); err != nil {
		if err == errNotSTUNMessage {
			return nil
		}
		log.Errorf("basicProcess: %v", err)
		return nil
	}
	_, err = c.WriteTo(res.Raw, addr)
	if err != nil {
		log.Errorf("WriteTo: %v", err)
	}
	return err
}

// Serve reads packets from connections and responds to BINDING requests.
func (s *Server) Serve(c net.PacketConn) error {
	var (
		res = new(stun.Message)
		req = new(stun.Message)
	)
	for {
		if err := s.serveConn(c, res, req); err != nil {
			log.Errorf("serve: %v", err)
			return err
		}
		res.Reset()
		req.Reset()
	}
}

// ListenUDPAndServe listens on laddr and process incoming packets.
func ListenUDPAndServe(serverNet, laddr string) error {
	c, err := net.ListenPacket(serverNet, laddr)
	if err != nil {
		return err
	}
	return new(Server).Serve(c)
}

func normalize(address string) string {
	if len(address) == 0 {
		address = "0.0.0.0"
	}
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:%d", address, stun.DefaultPort)
	}
	return address
}

func main() {
	if opts.Profile {
		go func() {
			log.Println(http.ListenAndServe(":6666", nil))
		}()
	}

	switch opts.Transport {
	case "udp":
		normalized := normalize(opts.Addr)
		log.Infoln("stuns listening on", normalized, "via", opts.Transport)
		log.Fatalln(ListenUDPAndServe(opts.Transport, normalized))
	default:
		log.Fatalln("Unsupported network:", opts.Transport)
	}
}
