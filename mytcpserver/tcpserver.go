package tcpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Server struct
type Server struct {
	listenAddr           *net.TCPAddr
	listener             *net.TCPListener
	shutdown             bool
	shutdownDeadline     time.Time
	requestHandler       RequestHandlerFunc
	connectionCreator    ConnectionCreatorFunc
	ctx                  *context.Context
	activeConnections    int32
	maxAcceptConnections int32
	acceptedConnections  int32
	tlsConfig            *tls.Config
	tlsEnabled           bool
	listenConfig         *ListenConfig
	connWaitGroup        sync.WaitGroup
	connStructPool       sync.Pool
	loops                int
	allowThreadLocking   bool
	ballast              []byte
}

// Connection interface
type Connection interface {
	net.Conn
	GetNetConn() net.Conn
	GetServer() *Server
	GetClientAddr() *net.TCPAddr
	GetServerAddr() *net.TCPAddr
	GetStartTime() time.Time
	SetContext(ctx *context.Context)
	GetContext() *context.Context

	// used internally
	Start()
	Reset(netConn net.Conn)
	SetServer(server *Server)
}

// TCPConn struct implementing Connection and embedding net.Conn
type TCPConn struct {
	net.Conn
	server            *Server
	ctx               *context.Context
	ts                int64
	_cacheLinePadding [24]byte
}

// Listener config struct
type ListenConfig struct {
	lc net.ListenConfig
	// Enable/disable SO_REUSEPORT (requires Linux >=2.4)
	SocketReusePort bool
	// Enable/disable TCP_FASTOPEN (requires Linux >=3.7 or Windows 10, version 1607)
	// For Linux:
	// - see https://lwn.net/Articles/508865/
	// - enable with "echo 3 >/proc/sys/net/ipv4/tcp_fastopen" for client and server
	// For Windows:
	// - enable with "netsh int tcp set global fastopen=enabled"
	SocketFastOpen bool
	// Queue length for TCP_FASTOPEN (default 256)
	SocketFastOpenQueueLen int
	// Enable/disable TCP_DEFER_ACCEPT (requires Linux >=2.4)
	SocketDeferAccept bool
}

// Request handler function type
type RequestHandlerFunc func(conn Connection)

// Connection creator function
type ConnectionCreatorFunc func() Connection

var defaultListenConfig *ListenConfig = &ListenConfig{
	SocketReusePort: true,
}

// Creates a new server instance
func NewServer(listenAddr string) (*Server, error) {
	la, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("error resolving address '%s': %s", listenAddr, err)
	}
	var s *Server

	s = &Server{
		listenAddr:   la,
		listenConfig: defaultListenConfig,
		connStructPool: sync.Pool{
			New: func() interface{} {
				conn := s.connectionCreator()
				conn.SetServer(s)
				return conn
			},
		},
	}

	s.connectionCreator = func() Connection {
		return &TCPConn{}
	}

	s.SetBallast(20)

	return s, nil
}

// Sets TLS config but does not enable TLS yet. TLS can be either enabled
// by using server.ListenTLS() or later by using connection.StartTLS()
func (s *Server) SetTLSConfig(config *tls.Config) {
	s.tlsConfig = config
}

// Returns previously set TLS config
func (s *Server) GetTLSConfig() *tls.Config {
	return s.tlsConfig
}

// Enable TLS (use server.SetTLSConfig() first)
func (s *Server) EnableTLS() error {
	if s.GetTLSConfig() == nil {
		return fmt.Errorf("no TLS config set")
	}
	s.tlsEnabled = true
	return nil
}

// Sets listen config
func (s *Server) SetListenConfig(config *ListenConfig) {
	s.listenConfig = config
}

// Returns listen config
func (s *Server) GetListenConfig() *ListenConfig {
	return s.listenConfig
}

// Starts listening
func (s *Server) Listen() (err error) {
	network := "tcp4"
	if IsIPv6Addr(s.listenAddr) {
		network = "tcp6"
	}

	s.listenConfig.lc.Control = applyListenSocketOptions(s.listenConfig)
	l, err := s.listenConfig.lc.Listen(*s.GetContext(), network, s.listenAddr.String())
	if err != nil {
		return err
	}
	if tcpl, ok := l.(*net.TCPListener); ok {
		s.listener = tcpl
	} else {
		return fmt.Errorf("listener must be of type net.TCPListener")
	}

	return nil
}

// Starts listening using TLS
func (s *Server) ListenTLS() (err error) {
	err = s.EnableTLS()
	if err != nil {
		return err
	}
	return s.Listen()
}

// Sets maximum number of connections that are being accepted before the
// server automatically shutdowns
func (s *Server) SetMaxAcceptConnections(limit int32) {
	atomic.StoreInt32(&s.maxAcceptConnections, limit)
}

// Returns number of currently active connections
func (s *Server) GetActiveConnections() int32 {
	return s.activeConnections
}

// Returns number of accepted connections
func (s *Server) GetAcceptedConnections() int32 {
	return s.acceptedConnections
}

// Returns listening address
func (s *Server) GetListenAddr() *net.TCPAddr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr().(*net.TCPAddr)
}

// Gracefully shutdown server but wait no longer than d for active connections.
// Use d = 0 to wait indefinitely for active connections.
func (s *Server) Shutdown(d time.Duration) (err error) {
	s.shutdownDeadline = time.Time{}
	if d > 0 {
		s.shutdownDeadline = time.Now().Add(d)
	}
	s.shutdown = true
	err = s.listener.Close()
	if err != nil {
		return err
	}
	return nil
}

// Shutdown server immediately, do not wait for any connections
func (s *Server) Halt() (err error) {
	return s.Shutdown(-1 * time.Second)
}
