package redact

import (
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

func TestIsRedactStringDatabaseType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     bool
	}{
		{name: "varchar", typeName: "varchar", want: true},
		{name: "json", typeName: "JSON", want: true},
		{name: "bigint", typeName: "bigint", want: false},
		{name: "datetime", typeName: "datetime", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsRedactStringDatabaseType(test.typeName); got != test.want {
				t.Fatalf("IsRedactStringDatabaseType(%q) = %v, want %v", test.typeName, got, test.want)
			}
		})
	}
}

func TestFilterBaseAPIDocResponseFields(t *testing.T) {
	document := &adminv1.BaseApiDoc{
		Responses: []*adminv1.BaseApiDocResponse{{
			Body: &adminv1.BaseApiDocSchema{
				Name: "body",
				Type: "object",
				Ref:  "system.admin.v1.BaseUser",
				Children: []*adminv1.BaseApiDocSchema{
					{Name: "id", Type: "integer"},
					{Name: "name", Type: "string"},
					{Name: "profile", Type: "object", Children: []*adminv1.BaseApiDocSchema{
						{Name: "age", Type: "integer"},
						{Name: "nickname", Type: "string"},
					}},
				},
			},
		}},
	}

	FilterBaseAPIDocResponseFields(document)

	children := document.Responses[0].Body.Children
	if len(children) != 2 || children[0].Name != "name" || children[1].Name != "profile" {
		t.Fatalf("unexpected filtered response fields: %+v", children)
	}
	profileChildren := children[1].Children
	if len(profileChildren) != 1 || profileChildren[0].Name != "nickname" {
		t.Fatalf("unexpected filtered nested fields: %+v", profileChildren)
	}
}

func TestFilterBaseAPIDocResponseFieldsWithoutTenant(t *testing.T) {
	document := &adminv1.BaseApiDoc{
		Responses: []*adminv1.BaseApiDocResponse{{
			Body: &adminv1.BaseApiDocSchema{
				Name: "body",
				Type: "object",
				Ref:  "system.admin.v1.BaseArea",
				Children: []*adminv1.BaseApiDocSchema{
					{Name: "name", Type: "string"},
				},
			},
		}},
	}

	FilterBaseAPIDocResponseFields(document)

	if document.Responses[0].Body != nil {
		t.Fatalf("非租户响应不应返回可脱敏字段: %+v", document.Responses[0].Body)
	}
}
