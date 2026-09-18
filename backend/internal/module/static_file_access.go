package module

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

// protectStaticFileAccess 根据文件元数据保护 /data/ 静态文件访问。
func protectStaticFileAccess(
	handler http.Handler,
	baseFileRepo *data.BaseFileRepository,
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
	return newStaticFileAccessHandler(handler, findFile, authenticator, userToken)
}

// newStaticFileAccessHandler 创建可注入文件查询和认证器的静态资源保护 Handler。
func newStaticFileAccessHandler(
	handler http.Handler,
	findFile func(context.Context, string) (*models.BaseFile, error),
	authenticator engine.TokenAuthenticator,
	userToken *authData.UserToken,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/data/") {
			handler.ServeHTTP(writer, request)
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
		if file.AccessMode == int32(basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_PUBLIC) {
			handler.ServeHTTP(writer, request)
			return
		}
		if !authorizeStaticFileRequest(request, file, authenticator, userToken) {
			writer.Header().Set("WWW-Authenticate", `Bearer realm="file"`)
			http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(writer, request)
	})
}

// authorizeStaticFileRequest 校验授权文件的 Bearer Token 和租户归属。
func authorizeStaticFileRequest(
	request *http.Request,
	file *models.BaseFile,
	authenticator engine.TokenAuthenticator,
	userToken *authData.UserToken,
) bool {
	if authenticator == nil || userToken == nil {
		return false
	}
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], engine.BearerWord) {
		return false
	}
	claims, err := authenticator.AuthenticateToken(parts[1])
	if err != nil {
		return false
	}
	authInfo, err := authData.NewUserTokenPayloadWithClaims(claims)
	if err != nil || authInfo.UserId <= 0 || !userToken.IsAccessTokenValid(authInfo.UserId, parts[1]) {
		return false
	}
	return file.TenantID == 0 || file.TenantID == authInfo.TenantId
}

var errStaticFileMetadataUnavailable = errors.New("文件元数据仓储未配置")
