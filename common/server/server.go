package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

var (
	GlobalHttpServer = NewHttpServer()
)

type HttpServer struct {
	gin                 *gin.Engine
	hostHandlerMapper   map[string]gin.HandlerFunc
	headerHandlerMapper map[string]gin.HandlerFunc
}

func NewHttpServer() (server *HttpServer) {
	server = &HttpServer{
		gin:                 gin.New(),
		hostHandlerMapper:   make(map[string]gin.HandlerFunc),
		headerHandlerMapper: make(map[string]gin.HandlerFunc),
	}
	server.Init()
	return server
}

func (s *HttpServer) Init() {
	s.gin.Handle(http.MethodGet, "/", s.CommonDispatchHandler)
	s.gin.Handle(http.MethodDelete, "/", s.CommonDispatchHandler)
	s.gin.Handle(http.MethodPut, "/", s.CommonDispatchHandler)
	s.gin.Handle(http.MethodPost, "/", s.CommonDispatchHandler)
}

func (s *HttpServer) CommonDispatchHandler(c *gin.Context) {
	if handler, ok := s.hostHandlerMapper[c.Request.Host]; ok {
		zap.L().Debug(fmt.Sprintf("match host: %s", c.Request.Host))
		handler(c)
		return
	}
	for key, values := range c.Request.Header {
		compare := fmt.Sprintf("%s=%s", key, strings.Join(values, ","))
		if handler, ok := s.headerHandlerMapper[compare]; ok {
			zap.L().Debug(fmt.Sprintf("match header: %s", compare))
			handler(c)
			return
		}
	}
}

func (s *HttpServer) AddOrReplaceHostMap(host string, handler gin.HandlerFunc) error {
	if _, exist := s.hostHandlerMapper[host]; exist {
		return fmt.Errorf("host handler map exist")
	}
	s.hostHandlerMapper[host] = handler
	return nil
}

func (s *HttpServer) DeleteHostMap(host string) {
	delete(s.hostHandlerMapper, host)
}

func (s *HttpServer) AddOrReplaceHeaderMap(header string, handler gin.HandlerFunc) error {
	if _, exist := s.headerHandlerMapper[header]; exist {
		return fmt.Errorf("header handler map exist")
	}
	s.headerHandlerMapper[header] = handler
	return nil
}

func (s *HttpServer) DeleteHeaderMap(header string) {
	delete(s.headerHandlerMapper, header)
}

func (s *HttpServer) Run(addr string) error {
	return s.gin.Run(addr)
}
