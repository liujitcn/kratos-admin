package projectauth

import (
	"encoding/json"
	"slices"

	"github.com/liujitcn/kratos-core/errorsx"
)

// Normalize 校验并去重项目授权，[0]只能单独表示全部项目。
func Normalize(ids []int64) ([]int64, error) {
	result := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id < 0 || id > 9007199254740991 || (id == 0 && len(ids) != 1) {
			return nil, errorsx.InvalidArgument("项目授权必须为安全正整数数组，全部授权只能使用[0]")
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	slices.Sort(result)
	return result, nil
}

// Decode 解析数据库中的数字数组，非法或缺失值不得退化为全部授权。
func Decode(value string) ([]int64, error) {
	var ids []int64
	err := json.Unmarshal([]byte(value), &ids)
	if err != nil || ids == nil {
		return nil, errorsx.Internal("项目授权数据格式无效")
	}
	return Normalize(ids)
}

// Allows 判断授权数组是否包含目标项目，项目ID必须为正数。
func Allows(ids []int64, projectID int64) bool {
	return projectID > 0 && (slices.Equal(ids, []int64{0}) || slices.Contains(ids, projectID))
}

// Merge 合并同一目标租户的多个授权来源，全部授权优先，空数组不抵消其他来源。
func Merge(groups ...[]int64) ([]int64, error) {
	result := make([]int64, 0)
	all := false
	for _, group := range groups {
		ids, err := Normalize(group)
		if err != nil {
			return nil, err
		}
		if slices.Equal(ids, []int64{0}) {
			all = true
			continue
		}
		result = append(result, ids...)
	}
	if all {
		return []int64{0}, nil
	}
	return Normalize(result)
}
