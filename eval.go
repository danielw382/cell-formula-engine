package formula

import (
	"fmt"
	"strconv"
	"strings"
)

// Sheet holds known cell values by reference, e.g. "A1" -> 12.5.
type Sheet map[string]float64

type valueKind int

const (
	kindNumber valueKind = iota
	kindBool
	kindString
)

// Value is the result of evaluating a formula. Arithmetic always produces a
// number; comparison operators (=, <>, <, >, <=, >=) and the TRUE/FALSE
// literals produce a bool instead, and string literals produce a string, so
// the result can't just be a float64.
type Value struct {
	kind valueKind
	num  float64
	b    bool
	str  string
}

func numberValue(n float64) Value { return Value{kind: kindNumber, num: n} }
func boolValue(b bool) Value      { return Value{kind: kindBool, b: b} }
func stringValue(s string) Value  { return Value{kind: kindString, str: s} }

// IsBool reports whether v holds a boolean rather than a number or string.
func (v Value) IsBool() bool { return v.kind == kindBool }

// IsString reports whether v holds a string rather than a number or bool.
func (v Value) IsString() bool { return v.kind == kindString }

// Number returns v's numeric value and true, or 0 and false if v holds a
// bool or string instead.
func (v Value) Number() (float64, bool) { return v.num, v.kind == kindNumber }

// Bool returns v's boolean value and true, or false and false if v holds a
// number or string instead.
func (v Value) Bool() (bool, bool) { return v.b, v.kind == kindBool }

// Text returns v's string value and true, or "" and false if v holds a
// number or bool instead.
func (v Value) Text() (string, bool) { return v.str, v.kind == kindString }

func (v Value) String() string {
	switch v.kind {
	case kindBool:
		if v.b {
			return "TRUE"
		}
		return "FALSE"
	case kindString:
		return v.str
	default:
		return strconv.FormatFloat(v.num, 'g', -1, 64)
	}
}

// Evaluate parses and computes a formula against sheet. A leading "=" is optional.
func Evaluate(formula string, sheet Sheet) (Value, error) {
	src := strings.TrimPrefix(strings.TrimSpace(formula), "=")
	n, err := parseExpr(src)
	if err != nil {
		return Value{}, err
	}
	return eval(n, sheet)
}

func eval(n node, sheet Sheet) (Value, error) {
	switch v := n.(type) {
	case numberNode:
		return numberValue(v.val), nil
	case boolNode:
		return boolValue(v.val), nil
	case stringNode:
		return stringValue(v.val), nil
	case cellNode:
		val, ok := sheet[v.ref]
		if !ok {
			return Value{}, fmt.Errorf("formula: cell %s has no value", v.ref)
		}
		return numberValue(val), nil
	case unaryNode:
		operand, err := eval(v.operand, sheet)
		if err != nil {
			return Value{}, err
		}
		num, ok := operand.Number()
		if !ok {
			return Value{}, fmt.Errorf("formula: %s is not a number", operand)
		}
		if v.op == tokMinus {
			return numberValue(-num), nil
		}
		return numberValue(num), nil
	case binaryNode:
		return evalBinary(v, sheet)
	case callNode:
		return evalCall(v, sheet)
	case rangeNode:
		return Value{}, fmt.Errorf("formula: range %s:%s can only be used as a function argument", v.from, v.to)
	default:
		return Value{}, fmt.Errorf("formula: unknown expression")
	}
}

func evalBinary(v binaryNode, sheet Sheet) (Value, error) {
	left, err := eval(v.left, sheet)
	if err != nil {
		return Value{}, err
	}
	right, err := eval(v.right, sheet)
	if err != nil {
		return Value{}, err
	}
	switch v.op {
	case tokPlus, tokMinus, tokStar, tokSlash:
		ln, ok := left.Number()
		if !ok {
			return Value{}, fmt.Errorf("formula: %s is not a number", left)
		}
		rn, ok := right.Number()
		if !ok {
			return Value{}, fmt.Errorf("formula: %s is not a number", right)
		}
		return evalArith(v.op, ln, rn)
	case tokEq:
		return boolValue(valuesEqual(left, right)), nil
	case tokNe:
		return boolValue(!valuesEqual(left, right)), nil
	case tokLt, tokLe, tokGt, tokGe:
		ln, ok := left.Number()
		if !ok {
			return Value{}, fmt.Errorf("formula: %s is not a number", left)
		}
		rn, ok := right.Number()
		if !ok {
			return Value{}, fmt.Errorf("formula: %s is not a number", right)
		}
		return evalCompare(v.op, ln, rn), nil
	default:
		return Value{}, fmt.Errorf("formula: unknown operator")
	}
}

func evalArith(op tokenKind, l, r float64) (Value, error) {
	switch op {
	case tokPlus:
		return numberValue(l + r), nil
	case tokMinus:
		return numberValue(l - r), nil
	case tokStar:
		return numberValue(l * r), nil
	case tokSlash:
		if r == 0 {
			return Value{}, fmt.Errorf("formula: division by zero")
		}
		return numberValue(l / r), nil
	default:
		return Value{}, fmt.Errorf("formula: unknown operator")
	}
}

func evalCompare(op tokenKind, l, r float64) Value {
	switch op {
	case tokLt:
		return boolValue(l < r)
	case tokLe:
		return boolValue(l <= r)
	case tokGt:
		return boolValue(l > r)
	default: // tokGe
		return boolValue(l >= r)
	}
}

// valuesEqual compares by kind first: a number, a bool, and a string are
// never equal to one another, the same way Excel treats 1 and TRUE as
// different for the = operator even though it happily coerces TRUE to 1 in
// arithmetic.
func valuesEqual(a, b Value) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case kindBool:
		return a.b == b.b
	case kindString:
		return a.str == b.str
	default:
		return a.num == b.num
	}
}

func evalCall(c callNode, sheet Sheet) (Value, error) {
	values, err := evalArgs(c.args, sheet)
	if err != nil {
		return Value{}, err
	}
	name := strings.ToUpper(c.name)
	if name == "COUNT" {
		return numberValue(float64(len(values))), nil
	}
	var total float64
	for _, v := range values {
		total += v
	}
	switch name {
	case "SUM":
		return numberValue(total), nil
	case "AVERAGE":
		if len(values) == 0 {
			return Value{}, fmt.Errorf("formula: AVERAGE needs at least one value")
		}
		return numberValue(total / float64(len(values))), nil
	case "MIN":
		if len(values) == 0 {
			return Value{}, fmt.Errorf("formula: MIN needs at least one value")
		}
		m := values[0]
		for _, v := range values[1:] {
			if v < m {
				m = v
			}
		}
		return numberValue(m), nil
	case "MAX":
		if len(values) == 0 {
			return Value{}, fmt.Errorf("formula: MAX needs at least one value")
		}
		m := values[0]
		for _, v := range values[1:] {
			if v > m {
				m = v
			}
		}
		return numberValue(m), nil
	default:
		return Value{}, fmt.Errorf("formula: unknown function %s", c.name)
	}
}

// evalArgs flattens call arguments into a single list of values. Range
// arguments (A1:B3) expand into every cell in the rectangle; cells absent
// from sheet are skipped rather than treated as errors, matching how
// spreadsheets treat blank cells inside a range.
func evalArgs(args []node, sheet Sheet) ([]float64, error) {
	var values []float64
	for _, arg := range args {
		if r, ok := arg.(rangeNode); ok {
			vs, err := evalRange(r, sheet)
			if err != nil {
				return nil, err
			}
			values = append(values, vs...)
			continue
		}
		v, err := eval(arg, sheet)
		if err != nil {
			return nil, err
		}
		n, ok := v.Number()
		if !ok {
			return nil, fmt.Errorf("formula: %s is not a number", v)
		}
		values = append(values, n)
	}
	return values, nil
}

func evalRange(r rangeNode, sheet Sheet) ([]float64, error) {
	fromCol, fromRow, err := splitRef(r.from)
	if err != nil {
		return nil, err
	}
	toCol, toRow, err := splitRef(r.to)
	if err != nil {
		return nil, err
	}
	r1, err := strconv.Atoi(fromRow)
	if err != nil {
		return nil, fmt.Errorf("formula: %q is not a cell reference", r.from)
	}
	r2, err := strconv.Atoi(toRow)
	if err != nil {
		return nil, fmt.Errorf("formula: %q is not a cell reference", r.to)
	}
	c1, c2 := colIndex(fromCol), colIndex(toCol)
	if c1 > c2 {
		c1, c2 = c2, c1
	}
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	var values []float64
	for row := r1; row <= r2; row++ {
		for col := c1; col <= c2; col++ {
			ref := colName(col) + strconv.Itoa(row)
			if v, ok := sheet[ref]; ok {
				values = append(values, v)
			}
		}
	}
	return values, nil
}
