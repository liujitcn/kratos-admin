package module

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/oss"
)

// protectStaticFileAccess 根据文件元数据保护并代理 /data/ 文件访问。
func protectStaticFileAccess(
	handler http.Handler,
	baseFileRepo *data.BaseFileRepository,
	storage oss.OSS,
	authenticator engine.TokenAuthenticator,
	userToken *authData.UserToken,
) http.Handler {
	findFile := func(ctx context.Context, objectPath string) (*models.BaseFile, error) {
		if baseFileRepo == nil {
			return nil, errStaticFileMetadataUnavailable
		}
		query := baseFileRepo.Query(ctx).BaseFile
		return baseFileRepo.Find(ctx, repository.Where(query.LinkURL.Eq(objectPath)))
	}
	readFile := func(objectPath string) ([]byte, error) {
		if storage == nil {
			return nil, errStaticFileStorageUnavailable
		}
		return storage.GetFileByte(objectPath)
	}
	return newStaticFileAccessHandler(handler, findFile, readFile, authenticator, userToken)
}

// newStaticFileAccessHandler 创建可注入文件查询、读取和认证器的资源 Handler。
func newStaticFileAccessHandler(
	handler http.Handler,
	findFile func(context.Context, string) (*models.BaseFile, error),
	readFile func(string) ([]byte, error),
	authenticator engine.TokenAuthenticator,
	userToken *authData.UserToken,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/data/") {
			handler.ServeHTTP(writer, request)
			return
		}
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", "GET, HEAD")
			http.Error(writer, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		objectPath, err := biz.ObjectFilePath(request.URL.Path)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		file, err := findFile(request.Context(), objectPath)
		if err != nil || file == nil {
			http.NotFound(writer, request)
			return
		}
		if file.AccessMode != int32(basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_PUBLIC) && !authorizeStaticFileRequest(request, file, authenticator, userToken) {
			writer.Header().Set("WWW-Authenticate", `Bearer realm="file"`)
			http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		content, err := readFile(objectPath)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		contentType := file.MimeType
		if contentType == "" {
			contentType = mime.TypeByExtension(filepath.Ext(file.FileName))
		}
		if contentType != "" {
			writer.Header().Set("Content-Type", contentType)
		}
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		if file.AccessMode == int32(basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_PUBLIC) {
			writer.Header().Set("Cache-Control", "public, max-age=3600")
		} else {
			writer.Header().Set("Cache-Control", "private, no-store")
		}
		modifiedAt := file.UpdatedAt
		if modifiedAt.IsZero() {
			modifiedAt = file.CreatedAt
		}
		http.ServeContent(writer, request, file.FileName, modifiedAt, bytes.NewReader(content))
	})
}

// authorizeStaticFileRequest 校验授权文件的请求头或 Cookie 访问令牌和租户归属。
func authorizeStaticFileRequest(
	request *http.Request,
	file *models.BaseFile,
	authenticator engine.TokenAuthenticator,
	userToken *authData.UserToken,
) bool {
	if authenticator == nil || userToken == nil {
		return false
	}
	token := staticFileAccessToken(request)
	if token == "" {
		return false
	}
	claims, err := authenticator.AuthenticateToken(token)
	if err != nil {
		return false
	}
	authInfo, err := authData.NewUserTokenPayloadWithClaims(claims)
	if err != nil || authInfo.UserId <= 0 || !userToken.IsAccessTokenValid(authInfo.UserId, token) {
		return false
	}
	return file.TenantID == 0 || file.TenantID == authInfo.TenantId
}

// staticFileAccessToken 优先读取请求头令牌，并在原生 src 请求中回退到访问令牌 Cookie。
func staticFileAccessToken(request *http.Request) string {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], engine.BearerWord) {
		return parts[1]
	}
	cookie, err := request.Cookie(staticFileAccessTokenCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

const staticFileAccessTokenCookieName = "kratos_access_token"

var (
	errStaticFileMetadataUnavailable = errors.New("文件元数据仓储未配置")
	errStaticFileStorageUnavailable  = errors.New("对象存储未配置")
)
