package httpserver

import (
	"context"
	"net"
	"net/http"
	"time"
)

type Config struct {
	Host              string
	Port              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func (c *Config) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

type Server struct {
	server *http.Server
}

func New(
	handler http.Handler,
	cfg Config,
) *Server {
	return &Server{
		server: &http.Server{
			Addr:              cfg.Address(),
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.server.Close()
}

func (s *Server) Address() string {
	return s.server.Addr
}
