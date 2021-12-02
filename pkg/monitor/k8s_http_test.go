package monitor

import (
	"eventrigger.com/operator/common/server"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httputil"
	"testing"
)

func ReverseHandler(c *gin.Context) (code int, data interface{}, err error){
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "www.baidu.com"
	}
	fmt.Printf("reverse to host %s \n", c.Request.Host)
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)
	return
}

func TestRedirectRequest(t *testing.T) {
	srv := server.NewHttpServer()
	srv.AddOrReplaceHostMap("www.baidu.com", ReverseHandler)
	srv.Run(":8081")
}
