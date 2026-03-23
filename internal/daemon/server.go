package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/canaanyjn/flarness/internal/model"
)

// Server is the Unix Domain Socket IPC server that listens for CLI commands.
type Server struct {
	socketPath string
	listener   net.Listener
	daemon     *Daemon
	handler    *Handler
	quit       chan struct{}
	wg         sync.WaitGroup
}

// NewServer creates a new IPC server.
func NewServer(socketPath string, d *Daemon) *Server {
	return &Server{
		socketPath: socketPath,
		daemon:     d,
		handler:    NewHandler(d),
		quit:       make(chan struct{}),
	}
}

// ListenAndServe starts the Unix socket listener and blocks until shutdown.
func (s *Server) ListenAndServe() error {
	var err error
	s.listener, err = net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.socketPath, err)
	}
	defer s.listener.Close()

	// Ensure socket is world-readable.
	os.Chmod(s.socketPath, 0666)

	// Handle OS signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "[flarness] received shutdown signal")
		s.Shutdown()
	}()

	fmt.Fprintf(os.Stderr, "[flarness] IPC server listening on %s\n", s.socketPath)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				// Normal shutdown.
				s.wg.Wait()
				return nil
			default:
				fmt.Fprintf(os.Stderr, "[flarness] accept error: %v\n", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection processes a single CLI connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var cmd model.Command
	if err := decoder.Decode(&cmd); err != nil {
		resp := model.Response{OK: false, Error: "invalid command: " + err.Error()}
		encoder.Encode(resp)
		return
	}

	resp := s.handler.Handle(cmd)
	encoder.Encode(resp)

	// If the command was "stop", trigger shutdown after responding.
	if cmd.Cmd == "stop" {
		go s.Shutdown()
	}
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() {
	select {
	case <-s.quit:
		// Already shutting down.
		return
	default:
		close(s.quit)
		s.listener.Close()
	}
}
