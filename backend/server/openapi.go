package server

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

const defaultOpenAPIPath = "/api/docs/openapi"

// mergeOpenAPIDocuments 按输入顺序合并多份 OpenAPI 文档，并保留业务文档优先的节点顺序。
func mergeOpenAPIDocuments(documents ...[]byte) ([]byte, error) {
	if len(documents) == 0 {
		return nil, fmt.Errorf("OpenAPI 文档不能为空")
	}

	var merged yaml.Node
	var err error
	for index, data := range documents {
		if len(data) == 0 {
			return nil, fmt.Errorf("第 %d 份 OpenAPI 文档为空", index+1)
		}

		var document yaml.Node
		err = yaml.Unmarshal(data, &document)
		if err != nil {
			return nil, fmt.Errorf("解析第 %d 份 OpenAPI 文档失败: %w", index+1, err)
		}
		var root *yaml.Node
		root, err = openAPIRoot(&document)
		if err != nil {
			return nil, fmt.Errorf("第 %d 份 OpenAPI 文档格式无效: %w", index+1, err)
		}

		if index == 0 {
			merged = document
			continue
		}
		var mergedRoot *yaml.Node
		mergedRoot, err = openAPIRoot(&merged)
		if err != nil {
			return nil, fmt.Errorf("合并 OpenAPI 文档格式无效: %w", err)
		}
		err = mergeOpenAPIRoot(mergedRoot, root)
		if err != nil {
			return nil, fmt.Errorf("合并第 %d 份 OpenAPI 文档失败: %w", index+1, err)
		}
	}

	var data []byte
	data, err = yaml.Marshal(&merged)
	if err != nil {
		return nil, fmt.Errorf("序列化合并后的 OpenAPI 文档失败: %w", err)
	}
	return data, nil
}

// openAPIRoot 获取 OpenAPI 文档的根映射节点。
func openAPIRoot(document *yaml.Node) (*yaml.Node, error) {
	if document == nil || document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return nil, fmt.Errorf("根节点必须是单一文档节点")
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("根节点必须是映射节点")
	}
	return root, nil
}

// mergeOpenAPIRoot 合并 OpenAPI 根节点中的可组合字段。
func mergeOpenAPIRoot(destination, source *yaml.Node) error {
	if destination.Kind != yaml.MappingNode || source.Kind != yaml.MappingNode {
		return fmt.Errorf("根节点必须是映射节点")
	}

	var err error
	for index := 0; index < len(source.Content); index += 2 {
		key := source.Content[index]
		value := source.Content[index+1]
		existing, found := openAPIMapValue(destination, key.Value)
		if !found {
			destination.Content = append(destination.Content, key, value)
			continue
		}

		switch key.Value {
		case "paths":
			err = mergeOpenAPIPaths(existing, value)
		case "components":
			err = mergeOpenAPIComponents(existing, value)
		case "tags", "servers", "security":
			err = mergeOpenAPISequence(existing, value, key.Value)
		default:
			if !yamlNodeEqual(existing, value) {
				err = fmt.Errorf("顶层字段 %q 内容冲突", key.Value)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// mergeOpenAPIPaths 合并 paths 映射，并允许同一路径补充不同 HTTP 操作。
func mergeOpenAPIPaths(destination, source *yaml.Node) error {
	if destination.Kind != yaml.MappingNode || source.Kind != yaml.MappingNode {
		return fmt.Errorf("paths 必须是映射节点")
	}

	var err error
	for index := 0; index < len(source.Content); index += 2 {
		key := source.Content[index]
		value := source.Content[index+1]
		existing, found := openAPIMapValue(destination, key.Value)
		if !found {
			destination.Content = append(destination.Content, key, value)
			continue
		}
		err = mergeOpenAPIPathItem(existing, value, key.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// mergeOpenAPIPathItem 合并同一路径下的操作定义，重复操作必须内容一致。
func mergeOpenAPIPathItem(destination, source *yaml.Node, path string) error {
	if destination.Kind != yaml.MappingNode || source.Kind != yaml.MappingNode {
		if yamlNodeEqual(destination, source) {
			return nil
		}
		return fmt.Errorf("路径 %q 内容冲突", path)
	}

	for index := 0; index < len(source.Content); index += 2 {
		key := source.Content[index]
		value := source.Content[index+1]
		existing, found := openAPIMapValue(destination, key.Value)
		if !found {
			destination.Content = append(destination.Content, key, value)
			continue
		}
		if !yamlNodeEqual(existing, value) {
			return fmt.Errorf("路径 %q 的操作 %q 内容冲突", path, key.Value)
		}
	}
	return nil
}

// mergeOpenAPIComponents 合并 components 下各类具名组件，重复组件必须内容一致。
func mergeOpenAPIComponents(destination, source *yaml.Node) error {
	if destination.Kind != yaml.MappingNode || source.Kind != yaml.MappingNode {
		return fmt.Errorf("components 必须是映射节点")
	}

	var err error
	for index := 0; index < len(source.Content); index += 2 {
		key := source.Content[index]
		value := source.Content[index+1]
		existing, found := openAPIMapValue(destination, key.Value)
		if !found {
			destination.Content = append(destination.Content, key, value)
			continue
		}
		err = mergeOpenAPINamedMapping(existing, value, "components."+key.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// mergeOpenAPINamedMapping 合并由名称索引的 OpenAPI 节点。
func mergeOpenAPINamedMapping(destination, source *yaml.Node, field string) error {
	if destination.Kind != yaml.MappingNode || source.Kind != yaml.MappingNode {
		return fmt.Errorf("%s 必须是映射节点", field)
	}

	for index := 0; index < len(source.Content); index += 2 {
		key := source.Content[index]
		value := source.Content[index+1]
		existing, found := openAPIMapValue(destination, key.Value)
		if !found {
			destination.Content = append(destination.Content, key, value)
			continue
		}
		if !yamlNodeEqual(existing, value) {
			return fmt.Errorf("%s.%s 内容冲突", field, key.Value)
		}
	}
	return nil
}

// mergeOpenAPISequence 追加 tags、servers 或 security 等顺序敏感的数组字段。
func mergeOpenAPISequence(destination, source *yaml.Node, field string) error {
	if destination.Kind != yaml.SequenceNode || source.Kind != yaml.SequenceNode {
		return fmt.Errorf("%s 必须是序列节点", field)
	}
	destination.Content = append(destination.Content, source.Content...)
	return nil
}

// openAPIMapValue 查找映射节点中指定名称对应的值节点。
func openAPIMapValue(mapping *yaml.Node, name string) (*yaml.Node, bool) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, false
	}
	for index := 0; index < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == name {
			return mapping.Content[index+1], true
		}
	}
	return nil, false
}

// yamlNodeEqual 比较两个 YAML 节点的语义内容，忽略格式、注释和原始位置差异。
func yamlNodeEqual(left, right *yaml.Node) bool {
	return reflect.DeepEqual(normalizeYAMLNode(left), normalizeYAMLNode(right))
}

// normalizeYAMLNode 将 YAML 节点转换为可比较的语义值。
func normalizeYAMLNode(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil
		}
		return normalizeYAMLNode(node.Content[0])
	case yaml.MappingNode:
		values := make(map[string]any, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			values[node.Content[index].Value] = normalizeYAMLNode(node.Content[index+1])
		}
		return values
	case yaml.SequenceNode:
		values := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			values = append(values, normalizeYAMLNode(child))
		}
		return values
	case yaml.ScalarNode:
		return struct {
			Tag   string
			Value string
		}{Tag: node.Tag, Value: node.Value}
	case yaml.AliasNode:
		return normalizeYAMLNode(node.Alias)
	default:
		return nil
	}
}
