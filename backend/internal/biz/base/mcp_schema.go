package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// agentSchemaTool 保留内置工具执行器，仅覆盖本次调用使用的有效输入契约。
type agentSchemaTool struct {
	tool.InvokableTool
	info *schema.ToolInfo
}

// Info 返回从 Proto 收紧后的工具输入契约。
func (t *agentSchemaTool) Info(context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

// agentToolWithSchema 复用内置执行器并发布冻结快照中的契约，不改写共享工具对象。
func agentToolWithSchema(ctx context.Context, value tool.BaseTool, raw string) (tool.BaseTool, error) {
	invokable, ok := value.(tool.InvokableTool)
	if !ok {
		return nil, errorsx.Internal("内置工具执行器类型无效")
	}
	info, err := value.Info(ctx)
	if err != nil {
		return nil, err
	}
	var input jsonschema.Schema
	if err = json.Unmarshal([]byte(raw), &input); err != nil {
		return nil, err
	}
	copied := *info
	copied.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(&input)
	return &agentSchemaTool{InvokableTool: invokable, info: &copied}, nil
}

// agentToolRequestDescriptor 从接口操作名取得唯一的 Proto 请求契约。
func agentToolRequestDescriptor(operation string) (protoreflect.MessageDescriptor, error) {
	operation = strings.ReplaceAll(strings.TrimPrefix(operation, "/"), "/", ".")
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(operation))
	if err != nil {
		return nil, errorsx.Internal("工具接口契约不存在").WithCause(err)
	}
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, errorsx.Internal("工具接口契约无效")
	}
	return method.Input(), nil
}

// agentToolSchema 用 Proto 枚举约束反射得到的工具 Schema，避免向模型声明任意字符串均合法。
func agentToolSchema(operation string, raw []byte) ([]byte, error) {
	descriptor, err := agentToolRequestDescriptor(operation)
	if err != nil {
		return nil, err
	}
	var value jsonschema.Schema
	if err = json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	constrainAgentToolEnums(&value, descriptor)
	return json.Marshal(&value)
}

// constrainAgentToolEnums 递归约束消息、列表和映射中的枚举，保持工具声明与 ProtoJSON 一致。
func constrainAgentToolEnums(value *jsonschema.Schema, descriptor protoreflect.MessageDescriptor) {
	if value == nil || value.Properties == nil {
		return
	}
	fields := descriptor.Fields()
	for index := range fields.Len() {
		field := fields.Get(index)
		property, ok := value.Properties.Get(string(field.Name()))
		if !ok {
			property, ok = value.Properties.Get(field.JSONName())
		}
		if !ok {
			continue
		}
		if field.IsMap() {
			property = property.AdditionalProperties
			field = field.MapValue()
		} else if field.IsList() {
			property = property.Items
		}
		if property == nil {
			continue
		}
		if field.Kind() == protoreflect.EnumKind {
			values := field.Enum().Values()
			names := make([]any, 0, values.Len())
			numbers := make([]any, 0, values.Len())
			for enumIndex := range values.Len() {
				item := values.Get(enumIndex)
				names = append(names, string(item.Name()))
				numbers = append(numbers, int32(item.Number()))
			}
			property.Type = ""
			property.TypeEnhanced = nil
			property.OneOf = []*jsonschema.Schema{{Type: "string", Enum: names}, {Type: "integer", Enum: numbers}}
		} else if field.Kind() == protoreflect.MessageKind {
			constrainAgentToolEnums(property, field.Message())
		}
	}
}
