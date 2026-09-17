package redact

import (
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// FilterBaseAPIDocResponseFields 过滤接口文档响应中的非字符串叶子字段。
func FilterBaseAPIDocResponseFields(document *adminv1.BaseApiDoc) {
	if document == nil {
		return
	}
	for _, response := range document.Responses {
		if response == nil {
			continue
		}
		response.Body = filterBaseAPIDocStringSchema(response.Body, true)
	}
}

// filterBaseAPIDocStringSchema 保留根节点和包含字符串后代的容器节点。
func filterBaseAPIDocStringSchema(schema *adminv1.BaseApiDocSchema, root bool) *adminv1.BaseApiDocSchema {
	if schema == nil {
		return nil
	}
	if len(schema.Children) == 0 {
		if root || strings.EqualFold(schema.Type, "string") {
			return schema
		}
		return nil
	}
	children := make([]*adminv1.BaseApiDocSchema, 0, len(schema.Children))
	for _, child := range schema.Children {
		filtered := filterBaseAPIDocStringSchema(child, false)
		if filtered != nil {
			children = append(children, filtered)
		}
	}
	schema.Children = children
	if !root && len(children) == 0 {
		return nil
	}
	return schema
}
