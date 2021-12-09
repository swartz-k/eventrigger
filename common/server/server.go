package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

var (
	GlobalHttpServer = NewHttpServer()
)

type HttpServer struct {
	gin                 *gin.Engine
	hostHandlerMapper   map[string]Handler
	headerHandlerMapper map[string]Handler
}

type HttpResponse struct {
	Code int `json:"code"`
	Msg string `json:"msg"`
	Data interface{} `json:"data"`
}

type Handler func(ctx *gin.Context) (int, interface{}, error)


func NewHttpServer() (server *HttpServer) {
	server = &HttpServer{
		gin:                 gin.New(),
		hostHandlerMapper:   make(map[string]Handler),
		headerHandlerMapper: make(map[string]Handler),
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
		zap.L().Info(fmt.Sprintf("match host: %s, handler %v", c.Request.Host, handler))
		var data interface{}
		code := 0
		err := errors.New("")
		code, data, err = handler(c)
		if err != nil {
			msg := HttpResponse{Msg:
				err.Error()}
			c.JSON(code, msg)
			return
		}
		if code == 0 {
			code = http.StatusOK
		}
		if data != nil {
			c.JSON(code, data)
			return
		}
		zap.L().Info(fmt.Sprintf("request proxy of host: %s done, data %s, code %d, err %+v",
			c.Request.Host, data, code, err))
		return
	}
	for key, values := range c.Request.Header {
		compare := fmt.Sprintf("%s=%s", key, strings.Join(values, ","))
		if handler, ok := s.headerHandlerMapper[compare]; ok {
			zap.L().Info(fmt.Sprintf("match header: %s,  handler %+v", compare, handler))
			code, data, err := handler(c)
			zap.L().Info(fmt.Sprintf("request proxy of header: %s done, data %s, code %d, err %+v",
				compare, data, code, err))
			return
		}
	}
	return
}

func (s *HttpServer) AddOrReplaceHostMap(host string, handler Handler) error {
	if _, exist := s.hostHandlerMapper[host]; exist {
		return fmt.Errorf("host handler map exist")
	}
	s.hostHandlerMapper[host] = handler
	return nil
}

func (s *HttpServer) DeleteHostMap(host string) {
	delete(s.hostHandlerMapper, host)
}

func (s *HttpServer) AddOrReplaceHeaderMap(header string, handler Handler) error {
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
