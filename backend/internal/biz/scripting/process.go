package scripting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-scripts/sandbox"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
	"github.com/liujitcn/kratos-kit/auth"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/transport/safehttp"
)

var scriptSlots = make(chan struct{}, 4)

// processRequest 只包含本次脚本所需数据，不复制主服务环境或基础设施对象。
type processRequest struct {
	Language string         `json:"language"`
	Source   string         `json:"source"`
	Input    map[string]any `json:"input"`
	Workflow bool           `json:"workflow"`
	Proxy    string         `json:"proxy"`
}

// processResponse 通过独立管道返回 JSON 结果，控制台不会参与协议解析。
type processResponse struct {
	Output json.RawMessage `json:"output,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// init 在宿主读取配置和建立数据库连接前进入隔离工作模式。
func init() {
	if sandbox.IsWorker() {
		os.Exit(runScriptWorker())
	}
}

// executeIsolated 在独立工作进程执行源码，统一限制并发、期限和输出。
func (r *Runtime) executeIsolated(ctx context.Context, workflow bool, language, source string, input map[string]any) (any, error) {
	timeout := r.timeout
	if timeout <= 0 || timeout > DefaultTimeout {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case scriptSlots <- struct{}{}:
		defer func() { <-scriptSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	directory, err := os.MkdirTemp("", "script-")
	if err != nil {
		return nil, err
	}
	defer func() {
		cleanupErr := os.RemoveAll(directory)
		if cleanupErr != nil {
			log.Error("清理脚本工作目录失败", "error", cleanupErr)
		}
	}()
	directory, err = filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, err
	}
	options := sandbox.Options{Directory: directory, MaxOutputBytes: 1024 * 1024}
	securityRuntimeConfig := securityconfig.Current()
	for _, item := range []struct {
		paths  []string
		target *[]string
	}{
		{paths: securityRuntimeConfig.ScriptReadPaths, target: &options.ReadOnlyPaths},
		{paths: securityRuntimeConfig.ScriptWritePaths, target: &options.WritablePaths},
		{paths: securityRuntimeConfig.ScriptCommands, target: &options.Commands},
	} {
		for _, path := range item.paths {
			// 路径来自部署配置，允许列表分隔符后的排版空白。
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			if strings.Contains(path, "{tenant_id}") || strings.Contains(path, "{user_id}") {
				var identity *authdata.UserTokenPayload
				identity, err = auth.FromContext(ctx)
				if err != nil || identity.UserId <= 0 || identity.TenantId <= 0 {
					return nil, fmt.Errorf("脚本资源路径需要有效的租户和用户身份")
				}
				path = strings.ReplaceAll(path, "{tenant_id}", strconv.FormatInt(identity.TenantId, 10))
				path = strings.ReplaceAll(path, "{user_id}", strconv.FormatInt(identity.UserId, 10))
			}
			*item.target = append(*item.target, path)
		}
	}
	var policy *safehttp.Policy
	policy, err = securityRuntimeConfig.ScriptPolicy()
	if err != nil {
		return nil, err
	}
	var listener net.Listener
	socket := filepath.Join(directory, ".proxy")
	listener, err = net.Listen("unix", socket)
	options.ReadOnlyPaths = append(options.ReadOnlyPaths, socket)
	if err != nil {
		return nil, err
	}
	proxy := &scriptProxy{policy: policy, ctx: ctx, transport: policy.Transport(http.DefaultTransport.(*http.Transport))}
	proxyClient := &http.Client{Transport: proxy.transport}
	defer proxyClient.CloseIdleConnections()
	server := &http.Server{Handler: proxy, BaseContext: func(net.Listener) context.Context { return ctx }}
	defer server.Close()
	go func() {
		serveErr := server.Serve(listener)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Error("脚本出站代理失败", "error", serveErr)
		}
	}()
	var request []byte
	request, err = json.Marshal(processRequest{Language: language, Source: source, Input: input, Workflow: workflow, Proxy: socket})
	if err != nil {
		return nil, err
	}
	var response []byte
	response, err = sandbox.Run(ctx, request, options)
	if err != nil {
		return nil, err
	}
	var result processResponse
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, fmt.Errorf("脚本工作进程响应无效: %w", err)
	}
	if result.Error != "" {
		return nil, errors.New(result.Error)
	}
	var output any
	decoder := json.NewDecoder(bytes.NewReader(result.Output))
	if !workflow {
		// 保留 Tcl/Go 等语言返回的整型精度，避免经过 float64 后损坏大整数。
		decoder.UseNumber()
	}
	err = decoder.Decode(&output)
	return output, err
}

// runScriptWorker 在系统沙箱中执行原有引擎流程，结果和错误沿用现有业务协议。
func runScriptWorker() (exitCode int) {
	response := processResponse{}
	defer func() {
		if failure := recover(); failure != nil {
			response = processResponse{Error: fmt.Sprintf("脚本执行失败: %v", failure)}
		}
		data, err := json.Marshal(response)
		if err == nil {
			err = sandbox.WriteResult(data)
		}
		if err != nil {
			exitCode = 1
		}
	}()
	err := sandbox.LimitWorker()
	if err != nil {
		response.Error = err.Error()
		return 0
	}
	var workingDirectory string
	workingDirectory, err = os.Getwd()
	if err != nil {
		response.Error = fmt.Sprintf("获取脚本工作目录失败: %v", err)
		return 0
	}
	err = os.Setenv("TMPDIR", workingDirectory)
	if err != nil {
		response.Error = fmt.Sprintf("设置脚本临时目录失败: %v", err)
		return 0
	}
	var request processRequest
	err = json.NewDecoder(io.LimitReader(os.Stdin, 4*1024*1024)).Decode(&request)
	if err != nil {
		response.Error = "脚本工作进程输入无效"
		return 0
	}
	// 系统沙箱阻断直接网络访问，只开放通过 Unix 套接字连接的受控 HTTP 代理。
	proxyURL := &url.URL{Scheme: "http", Host: "script.proxy"}
	http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyURL), DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", request.Proxy)
	}}
	runtime := NewRuntime()
	var output any
	if request.Workflow {
		output, err = runtime.ExecuteWorkflow(context.Background(), request.Language, request.Source, request.Input)
	} else {
		output, err = runtime.Execute(context.Background(), request.Language, request.Source, request.Input)
	}
	if err == nil {
		response.Output, err = json.Marshal(output)
	}
	if err != nil {
		response.Error = err.Error()
	}
	return 0
}
