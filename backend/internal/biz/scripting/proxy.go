package scripting

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/liujitcn/kratos-kit/transport/safehttp"
)

// scriptProxy 将脚本网络请求限制在服务端配置范围，工作进程不能直接连接外部网络。
type scriptProxy struct {
	policy    *safehttp.Policy
	ctx       context.Context
	transport http.RoundTripper
}

// ServeHTTP 转发已授权的 HTTP 或 HTTPS 隧道，不记录包含用户数据的请求内容。
func (p *scriptProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var err error
	if request.Method == http.MethodConnect {
		target := &url.URL{Scheme: "https", Host: request.Host}
		err = p.policy.ValidateURL(request.Context(), target)
		if err != nil {
			http.Error(writer, "脚本网络目标未授权", http.StatusForbidden)
			return
		}
		var upstream net.Conn
		upstream, err = p.policy.DialContext(request.Context(), "tcp", request.Host)
		if err != nil {
			http.Error(writer, "脚本网络连接失败", http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		var client net.Conn
		var buffered *bufio.ReadWriter
		client, buffered, err = writer.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer client.Close()
		stop := context.AfterFunc(p.ctx, func() {
			// 任务结束后必须关闭已劫持连接，HTTP Server.Close 不负责这些连接。
			_ = client.Close()
			_ = upstream.Close()
		})
		defer stop()
		_, err = buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		if err == nil {
			err = buffered.Flush()
		}
		if err != nil {
			return
		}
		go func() {
			// 连接关闭和超时属于正常隧道收尾，双方均由任务上下文负责关闭。
			_, _ = io.Copy(upstream, buffered)
		}()
		_, _ = io.Copy(client, upstream)
		return
	}
	forward := request.Clone(request.Context())
	forward.RequestURI = ""
	forward.Host = forward.URL.Host
	forward.Header.Del("Proxy-Authorization")
	forward.Header.Del("Proxy-Connection")
	var response *http.Response
	response, err = p.transport.RoundTrip(forward)
	if err != nil {
		http.Error(writer, "脚本网络目标未授权或连接失败", http.StatusForbidden)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.WriteHeader(response.StatusCode)
	// 客户端断开或超时由上层任务终止，响应体不进入审计日志。
	_, _ = io.Copy(writer, response.Body)
}
