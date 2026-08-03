package static

import (
	"fmt"
	"io/fs"
	stdhttp "net/http"
	"path"
	"strings"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
)

// Mount 描述一项静态资源挂载。
type Mount struct {
	// Prefix 是 HTTP 路由前缀。
	Prefix string
	// FS 是静态资源文件系统。
	FS fs.FS
	// SPA 表示找不到真实文件时是否回退到 index.html。
	SPA bool
}

// RegisterHTTP 校验并挂载静态资源，重复或重叠路由前缀会返回错误。
func RegisterHTTP(server *kratosHTTP.Server, mounts ...Mount) error {
	var err error
	prefixes := make(map[string]struct{}, len(mounts))
	for _, mount := range mounts {
		prefix := normalizePrefix(mount.Prefix)
		if prefix == "/" {
			return fmt.Errorf("静态资源路由前缀不能为根路径")
		}
		if mount.FS == nil {
			return fmt.Errorf("静态资源文件系统不能为空: %s", prefix)
		}
		for registeredPrefix := range prefixes {
			if prefix == registeredPrefix || strings.HasPrefix(prefix, registeredPrefix+"/") || strings.HasPrefix(registeredPrefix, prefix+"/") {
				return fmt.Errorf("静态资源路由前缀重叠: %s、%s", registeredPrefix, prefix)
			}
		}
		if mount.SPA {
			_, err = fs.Stat(mount.FS, "index.html")
			if err != nil {
				return fmt.Errorf("单页应用入口不存在 %s/index.html: %w", prefix, err)
			}
		}
		prefixes[prefix] = struct{}{}
	}
	for _, mount := range mounts {
		prefix := normalizePrefix(mount.Prefix)
		handler := newHandler(mount.FS, prefix, mount.SPA)
		if prefix == "/" {
			server.HandlePrefix(prefix, handler)
			continue
		}
		server.Handle(prefix, handler)
		server.HandlePrefix(prefix+"/", handler)
	}
	return nil
}

func newHandler(webFS fs.FS, prefix string, spa bool) stdhttp.Handler {
	fileHandler := stdhttp.Handler(stdhttp.FileServer(stdhttp.FS(webFS)))
	if prefix != "/" {
		fileHandler = stdhttp.StripPrefix(prefix, fileHandler)
	}
	if !spa {
		return fileHandler
	}
	return stdhttp.HandlerFunc(func(writer stdhttp.ResponseWriter, request *stdhttp.Request) {
		relativePath := strings.TrimPrefix(request.URL.Path, prefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		if relativePath == "" {
			stdhttp.ServeFileFS(writer, request, webFS, "index.html")
			return
		}
		var err error
		_, err = fs.Stat(webFS, relativePath)
		if err == nil {
			fileHandler.ServeHTTP(writer, request)
			return
		}
		stdhttp.ServeFileFS(writer, request, webFS, "index.html")
	})
}

func normalizePrefix(prefix string) string {
	prefix = "/" + strings.Trim(prefix, "/")
	return path.Clean(prefix)
}
