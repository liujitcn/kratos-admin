package server

import (
	"io/fs"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/internal/cmd/server/assets"
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	appMiddleware "github.com/liujitcn/kratos-admin/backend/pkg/middleware"
	"github.com/liujitcn/kratos-admin/backend/pkg/middleware/logging"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	kratosMiddleware "github.com/go-kratos/kratos/v3/middleware"
	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	"github.com/liujitcn/kratos-kit/rpc/middleware/requestid"
	swaggerUI "github.com/liujitcn/kratos-kit/swagger-ui"
)

// HTTPMiddlewares 表示 HTTP 服务中间件链。
type HTTPMiddlewares []kratosMiddleware.Middleware

// NewHTTPMiddleware 创建 HTTP 服务统一中间件链。
func NewHTTPMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepository,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConfigv1.Authentication_Jwt,
) HTTPMiddlewares {
	var ms HTTPMiddlewares
	cfg := ctx.GetConfig()
	// 先补齐请求标识，再进入访问日志中间件，确保日志能读取到统一 request_id。
	ms = append(ms, requestid.NewRequestIDMiddleware())
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Http != nil && cfg.Server.Http.Middleware != nil && cfg.Server.Http.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, appMiddleware.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	ms = append(ms, appMiddleware.NewValidateMiddleware())
	return ms
}

// NewHTTPServer 创建 HTTP Server 并注册已启用业务模块与前端静态路由。
func NewHTTPServer(
	ctx *bootstrap.Context,
	appInfo *bootstrapConfigv1.AppInfo,
	middlewares HTTPMiddlewares,
	modules Modules,
	authenticator authnEngine.Authenticator,
	userToken *authData.UserToken,
	_ MCPToolsReady,
	_ AgentToolsReady,
) (*kratosHTTP.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 HTTP 配置时，跳过 HTTP 服务创建。
	if cfg == nil || cfg.Server == nil || cfg.Server.Http == nil {
		return nil, nil
	}

	srv, err := rpc.CreateHttpServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	modules.RegisterHTTP(srv)

	ossRootDirectory := "./data"
	// 配置了本地 OSS 根目录时，优先使用配置值覆盖默认目录。
	if cfg.GetOss() != nil && cfg.GetOss().GetRootDirectory() != "" {
		ossRootDirectory = cfg.GetOss().GetRootDirectory()
	}
	projectName := appInfo.GetProject()
	staticPrefix := "/" + projectName + "/"
	staticDirectory := filepath.Join(ossRootDirectory, projectName)
	// 将本地 OSS 目录暴露为静态资源目录，默认访问 /admin/* 时映射到 ./data/admin/*。
	staticHandler := stdhttp.StripPrefix(staticPrefix, stdhttp.FileServer(stdhttp.Dir(staticDirectory)))
	srv.HandlePrefix(staticPrefix, staticHandler)

	// 自动发现本地 OSS 根目录下的前端入口，按子目录名称挂载为 SPA 路由。
	registerLocalSPARoutes(srv, ossRootDirectory)

	// 显式启用 Swagger 时，仅注册供管理后台加载的受保护 OpenAPI 文档接口。
	if cfg.GetServer().GetHttp().GetEnableSwagger() {
		swaggerUI.RegisterOpenAPIServerWithOption(
			srv,
			swaggerUI.WithOpenAPIPath("/api/docs/openapi"),
			swaggerUI.WithMemoryData(assets.OpenAPIData, "yaml"),
			swaggerUI.WithOpenAPIAuthorizer(newOpenAPIAuthorizer(authenticator, userToken)),
		)
	}

	return srv, nil
}

// registerLocalSPARoutes 扫描根目录下包含 index.html 的子目录，并按目录名注册单页应用路由。
func registerLocalSPARoutes(srv *kratosHTTP.Server, rootDirectory string) {
	entries, err := os.ReadDir(rootDirectory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		// 仅处理目录，忽略根目录下的普通文件。
		if !entry.IsDir() {
			continue
		}
		var directoryName = entry.Name()
		var indexPath = filepath.Join(rootDirectory, directoryName, "index.html")
		// 子目录未提供入口页面时，不注册为单页应用。
		if _, err = os.Stat(indexPath); err != nil {
			continue
		}
		var routePrefix = "/" + directoryName
		var spaHandler = newSPAHandler(os.DirFS(filepath.Join(rootDirectory, directoryName)), routePrefix)
		srv.Handle(routePrefix, spaHandler)
		srv.HandlePrefix(routePrefix+"/", spaHandler)
	}
}

// newSPAHandler 创建基于文件系统的单页应用处理器。
func newSPAHandler(webFS fs.FS, urlPrefix string) stdhttp.Handler {
	var fileHandler = stdhttp.StripPrefix(urlPrefix, stdhttp.FileServer(stdhttp.FS(webFS)))
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		var relativePath = strings.TrimPrefix(r.URL.Path, urlPrefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		// 访问应用根路径时，直接返回入口页面。
		if relativePath == "" {
			stdhttp.ServeFileFS(w, r, webFS, "index.html")
			return
		}
		// 命中真实静态文件时，交给文件服务直接返回。
		if _, err := fs.Stat(webFS, relativePath); err == nil {
			fileHandler.ServeHTTP(w, r)
			return
		}
		// 前端路由命中不到真实文件时，统一回退到入口页面。
		stdhttp.ServeFileFS(w, r, webFS, "index.html")
	})
}

// newOpenAPIAuthorizer 创建校验 Bearer Token 与服务端登录状态的 OpenAPI 访问校验函数。
func newOpenAPIAuthorizer(authenticator authnEngine.Authenticator, userToken *authData.UserToken) func(*stdhttp.Request) bool {
	return func(r *stdhttp.Request) bool {
		authorization := strings.Fields(r.Header.Get("Authorization"))
		if len(authorization) != 2 || !strings.EqualFold(authorization[0], authnEngine.BearerWord) {
			return false
		}

		claims, err := authenticator.AuthenticateToken(authorization[1])
		if err != nil {
			return false
		}

		var userID int64
		userID, err = claims.GetInt64(authData.ClaimFieldUserID)
		if err != nil {
			return false
		}
		return userID == 0 || userToken.IsExistAccessToken(userID)
	}
}
