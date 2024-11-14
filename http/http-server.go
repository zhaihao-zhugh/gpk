package http

import (
	"net/http"
	"strconv"
	"time"
)

var s *HttpServer

func init() {
	s = &HttpServer{
		&http.Server{
			ReadTimeout:    2 * time.Minute,
			WriteTimeout:   2 * time.Minute,
			MaxHeaderBytes: 1 << 20,
		},
	}
}
func Run(port int) error {
	s.SetAddress(":" + strconv.Itoa(port))
	return s.Run()
}

func SetHandler(handler http.Handler) {
	s.SetHandler(handler)
}

type HttpServer struct {
	*http.Server
}

func (s *HttpServer) SetAddress(addr string) {
	s.Addr = addr
}

func (s *HttpServer) SetHandler(handler http.Handler) {
	s.Handler = handler
}

func (s *HttpServer) Run() error {
	return s.ListenAndServe()
}
