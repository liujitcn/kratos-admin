package redact

import (
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
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
		response.Body = filterBaseAPIDocStringSchema(response.Body, false)
	}
}

// filterBaseAPIDocStringSchema 只保留带租户字段消息中的字符串叶子字段及其容器。
func filterBaseAPIDocStringSchema(schema *adminv1.BaseApiDocSchema, tenantScoped bool) *adminv1.BaseApiDocSchema {
	if schema == nil {
		return nil
	}
	if schemaRefContainsTenant(schema.Ref) {
		tenantScoped = true
	}
	if len(schema.Children) == 0 {
		if tenantScoped && strings.EqualFold(schema.Type, "string") {
			return schema
		}
		return nil
	}
	children := make([]*adminv1.BaseApiDocSchema, 0, len(schema.Children))
	for _, child := range schema.Children {
		filtered := filterBaseAPIDocStringSchema(child, tenantScoped)
		if filtered != nil {
			children = append(children, filtered)
		}
	}
	schema.Children = children
	if len(children) == 0 {
		return nil
	}
	return schema
}

// schemaRefContainsTenant 判断响应结构引用是否包含租户字段。
func schemaRefContainsTenant(reference string) bool {
	name := reference
	if index := strings.LastIndex(reference, "/"); index >= 0 {
		name = reference[index+1:]
	}
	if name == "" {
		return false
	}
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(name))
	if err != nil {
		return false
	}
	tenantField := messageType.Descriptor().Fields().ByName("tenant_id")
	return tenantField != nil && tenantField.Kind() == protoreflect.Int64Kind
}
