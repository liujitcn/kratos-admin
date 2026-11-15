package query

import (
	"fmt"
	"regexp"
	"strings"
	"text/scanner"
	"unicode/utf8"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

var decimalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// Expression 是经过封闭语法和数值类型检查的公共指标公式。
type Expression struct {
	root *expressionNode
}

// expressionNode 保存操作符、函数、常量或已验证的物理字段。
type expressionNode struct {
	kind     rune
	value    string
	children []*expressionNode
}

// expressionParser 限制词法、递归深度和节点数量。
type expressionParser struct {
	scan  scanner.Scanner
	token rune
	nodes int
	err   error
}

// Parse 解析只引用事实表字段的固定数值聚合表达式。
func Parse(input string, fields map[string]bi.FieldType) (*Expression, error) {
	if utf8.RuneCountInString(input) > 2048 {
		return nil, fmt.Errorf("指标公式最多2048字符")
	}
	p := &expressionParser{}
	p.scan.Init(strings.NewReader(input))
	p.scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats | scanner.ScanStrings
	p.scan.Error = func(_ *scanner.Scanner, message string) { p.err = fmt.Errorf("指标公式词法无效: %s", message) }
	p.token = p.scan.Scan()
	root := p.parseBinary(0, 0)
	if p.err != nil {
		return nil, p.err
	}
	if p.token != scanner.EOF {
		return nil, fmt.Errorf("指标公式含不支持的语法")
	}
	_, aggregated, err := validateExpression(root, fields, false, 0)
	if err != nil {
		return nil, err
	}
	if !aggregated {
		return nil, fmt.Errorf("指标公式必须包含聚合")
	}
	return &Expression{root: root}, nil
}

// Canonical 返回稳定的函数大小写和括号形式，不改变小数精度。
func (e *Expression) Canonical() string {
	return renderExpression(e.root, func(name string) string { return name }, false)
}

// SQL 只从已验证的表达式树生成 SQL，物理字段交给调用者按方言引用。
func (e *Expression) SQL(quote func(string) string) string {
	return renderExpression(e.root, quote, true)
}

// parseBinary 使用优先级解析四则运算，并限制递归。
func (p *expressionParser) parseBinary(minimum, depth int) *expressionNode {
	if depth > 32 || p.err != nil {
		p.err = fmt.Errorf("指标公式最多32层嵌套")
		return nil
	}
	left := p.parsePrimary(depth)
	for p.err == nil {
		precedence := 0
		switch p.token {
		case '+', '-':
			precedence = 1
		case '*', '/':
			precedence = 2
		}
		if precedence == 0 || precedence < minimum {
			break
		}
		operator := p.token
		p.token = p.scan.Scan()
		right := p.parseBinary(precedence+1, depth)
		left = p.node(operator, "", left, right)
	}
	return left
}

// parsePrimary 解析受支持的常量、字段、括号和函数调用。
func (p *expressionParser) parsePrimary(depth int) *expressionNode {
	if depth > 32 || p.err != nil {
		p.err = fmt.Errorf("指标公式最多32层嵌套")
		return nil
	}
	token, value := p.token, p.scan.TokenText()
	p.token = p.scan.Scan()
	switch token {
	case scanner.Int, scanner.Float:
		if !decimalPattern.MatchString(value) {
			p.err = fmt.Errorf("指标公式只支持十进制整数和小数")
		}
		return p.node(scanner.Int, value)
	case '+', '-':
		return p.node(token, "", p.parsePrimary(depth+1))
	case '(':
		node := p.parseBinary(0, depth+1)
		p.consume(')')
		return node
	case scanner.Ident:
		if strings.EqualFold(value, "NULL") {
			return p.node(scanner.Ident, "NULL")
		}
		if p.token != '(' {
			return p.node(scanner.Ident, value)
		}
		p.token = p.scan.Scan()
		args := make([]*expressionNode, 0, 2)
		if p.token == '*' {
			args = append(args, p.node('*', "*"))
			p.token = p.scan.Scan()
		} else {
			args = append(args, p.parseBinary(0, depth+1))
			for p.token == ',' && p.err == nil {
				p.token = p.scan.Scan()
				args = append(args, p.parseBinary(0, depth+1))
			}
		}
		p.consume(')')
		return p.node('f', strings.ToUpper(value), args...)
	default:
		p.err = fmt.Errorf("指标公式含不支持的语法")
		return nil
	}
}

// consume 验证固定分隔符，错误后停止继续扫描。
func (p *expressionParser) consume(token rune) {
	if p.err != nil {
		return
	}
	if p.token != token {
		p.err = fmt.Errorf("指标公式括号或参数无效")
		return
	}
	p.token = p.scan.Scan()
}

// node 统一限制公式大小，避免构造无限制语法树。
func (p *expressionParser) node(kind rune, value string, children ...*expressionNode) *expressionNode {
	p.nodes++
	if p.nodes > 512 {
		p.err = fmt.Errorf("指标公式最多512个节点")
	}
	return &expressionNode{kind: kind, value: value, children: children}
}

// validateExpression 检查聚合层级、字段作用域和每个参数的数值类型。
func validateExpression(node *expressionNode, fields map[string]bi.FieldType, inAggregate bool, depth int) (bool, bool, error) {
	var err error
	if depth > 32 {
		return false, false, fmt.Errorf("指标公式最多32层嵌套")
	}
	switch node.kind {
	case scanner.Int:
		return true, false, nil
	case scanner.Ident:
		if node.value == "NULL" {
			return true, false, nil
		}
		field, exists := fields[node.value]
		if !exists || !inAggregate {
			return false, false, fmt.Errorf("指标公式字段不存在或未被聚合: %s", node.value)
		}
		return field == bi.FieldTypeNumber, false, nil
	case 'f':
		switch node.value {
		case "COUNT", "SUM", "AVG", "MIN", "MAX":
			if inAggregate || len(node.children) != 1 {
				return false, false, fmt.Errorf("聚合不能嵌套且只接受一个参数")
			}
			argument := node.children[0]
			if node.value == "COUNT" {
				if argument.value == "*" {
					return true, true, nil
				}
				if argument.kind != scanner.Ident || argument.value == "NULL" {
					return false, false, fmt.Errorf("COUNT 只接受星号或事实表字段")
				}
				_, exists := fields[argument.value]
				if !exists {
					return false, false, fmt.Errorf("COUNT 字段不存在")
				}
				return true, true, nil
			}
			var numeric bool
			numeric, _, err = validateExpression(argument, fields, true, depth+1)
			if err != nil {
				return false, false, err
			}
			if !numeric {
				return false, false, fmt.Errorf("聚合参数必须是数值")
			}
			return true, true, nil
		case "COALESCE", "NULLIF":
			if len(node.children) < 2 || (node.value == "NULLIF" && len(node.children) != 2) {
				return false, false, fmt.Errorf("空值函数参数数量无效")
			}
		default:
			return false, false, fmt.Errorf("指标公式函数不支持: %s", node.value)
		}
	case '+', '-', '*', '/':
		if len(node.children) == 0 {
			return false, false, fmt.Errorf("星号只能用于 COUNT")
		}
	default:
		return false, false, fmt.Errorf("指标公式语法不支持")
	}
	aggregated := false
	for _, child := range node.children {
		var numeric, childAggregate bool
		numeric, childAggregate, err = validateExpression(child, fields, inAggregate, depth+1)
		if err != nil {
			return false, false, err
		}
		if !numeric {
			return false, false, fmt.Errorf("指标运算参数必须是数值")
		}
		aggregated = aggregated || childAggregate
	}
	return true, aggregated, nil
}

// renderExpression 从封闭节点渲染规范形式或带精确除法的执行形式。
func renderExpression(node *expressionNode, quote func(string) string, sql bool) string {
	if node.kind == scanner.Ident && node.value != "NULL" {
		return quote(node.value)
	}
	if len(node.children) == 0 {
		return node.value
	}
	args := make([]string, 0, len(node.children))
	for _, child := range node.children {
		args = append(args, renderExpression(child, quote, sql))
	}
	if node.kind == 'f' {
		return node.value + "(" + strings.Join(args, ", ") + ")"
	}
	if len(args) == 1 {
		return "(" + string(node.kind) + args[0] + ")"
	}
	if node.kind == '/' && sql {
		args[0] = "(" + args[0] + " * 1.0)"
	}
	return "(" + args[0] + " " + string(node.kind) + " " + args[1] + ")"
}
