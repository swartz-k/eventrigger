package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"net/http"
	"net/http/httputil"
)

func RedirectHttpRequest(c *gin.Context, pod *v1.Pod) (response []byte, err error) {
	if c == nil || pod == nil {
		return nil, errors.New("context or pod is nil")
	}
	var port int32
	for _, c := range pod.Spec.Containers {
		for _, p := range c.Ports {
			if port == 0 {
				port = p.HostPort
			}
			if p.Name == "http" {
				port = p.HostPort
			}
		}
	}
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("%s.%s:%d", pod.Name, pod.Namespace, port)
		req.URL.Path = c.Request.URL.Path
		// add custom headers
		req.Header["my-header"] = []string{req.Header.Get("my-header")}
		// delete Origin headers
		delete(req.Header, "My-Header")
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)
	return nil, err
}
