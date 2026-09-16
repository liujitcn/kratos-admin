package projectauth

import (
	"reflect"
	"testing"
)

// TestMerge 验证四维授权并集、全部优先和空集合语义。
func TestMerge(t *testing.T) {
	cases := []struct {
		name    string
		groups  [][]int64
		want    []int64
		invalid bool
	}{
		{name: "四维并集", groups: [][]int64{{101}, {102, 101}, {103}, {104}}, want: []int64{101, 102, 103, 104}},
		{name: "角色全部", groups: [][]int64{{101}, {0}, {103}, {}}, want: []int64{0}},
		{name: "空用户不抵消岗位", groups: [][]int64{{101}, {}, {}, {}}, want: []int64{101}},
		{name: "无授权", groups: [][]int64{{}, {}, {}, {}}, want: []int64{}},
		{name: "全部混用", groups: [][]int64{{0, 101}}, invalid: true},
		{name: "全部不能掩盖损坏来源", groups: [][]int64{{0}, {-1}}, invalid: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got, err := Merge(item.groups...)
			if (err != nil) != item.invalid {
				t.Fatalf("错误状态不符: %v", err)
			}
			if err == nil && !reflect.DeepEqual(got, item.want) {
				t.Fatalf("got %v want %v", got, item.want)
			}
		})
	}
}

// TestDecode 验证存储必须为整数数组，错误不能被视为全部授权。
func TestDecode(t *testing.T) {
	for _, value := range []string{"null", "{}", "[1.5]", `["101"]`, "[-1]", "[0,101]", "[9007199254740992]"} {
		if _, err := Decode(value); err == nil {
			t.Fatalf("应拒绝 %s", value)
		}
	}
	for _, value := range []string{"[]", "[0]", "[101,102]"} {
		if _, err := Decode(value); err != nil {
			t.Fatalf("应接受 %s: %v", value, err)
		}
	}
}
