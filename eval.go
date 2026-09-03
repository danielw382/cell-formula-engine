package formula

import (
	"fmt"
	"strconv"
	"strings"
)

// Sheet holds known cell values by reference, e.g. "A1" -> 12.5.
type Sheet map[string]float64

// Evaluate parses and computes a formula against sheet. A leading "=" is optional.
func Evaluate(formula string, sheet Sheet) (float64, error) {
	src := strings.TrimPrefix(strings.TrimSpace(formula), "=")
	n, err := parseExpr(src)
	if err != nil {
		return 0, err
	}
	return eval(n, sheet)
}

func eval(n node, sheet Sheet) (float64, error) {
	switch v := n.(type) {
	case numberNode:
		return v.val, nil
	case cellNode:
		val, ok := sheet[v.ref]
		if !ok {
			return 0, fmt.Errorf("formula: cell %s has no value", v.ref)
		}
		return val, nil
	case unaryNode:
		operand, err := eval(v.operand, sheet)
		if err != nil {
			return 0, err
		}
		if v.op == tokMinus {
			return -operand, nil
		}
		return operand, nil
	case binaryNode:
		return evalBinary(v, sheet)
	case callNode:
		return evalCall(v, sheet)
	case rangeNode:
		return 0, fmt.Errorf("formula: range %s:%s can only be used as a function argument", v.from, v.to)
	default:
		return 0, fmt.Errorf("formula: unknown expression")
	}
}

func evalBinary(v binaryNode, sheet Sheet) (float64, error) {
	left, err := eval(v.left, sheet)
	if err != nil {
		return 0, err
	}
	right, err := eval(v.right, sheet)
	if err != nil {
		return 0, err
	}
	switch v.op {
	case tokPlus:
		return left + right, nil
	case tokMinus:
		return left - right, nil
	case tokStar:
		return left * right, nil
	case tokSlash:
		if right == 0 {
			return 0, fmt.Errorf("formula: division by zero")
		}
		return left / right, nil
	default:
		return 0, fmt.Errorf("formula: unknown operator")
	}
}

func evalCall(c callNode, sheet Sheet) (float64, error) {
	values, err := evalArgs(c.args, sheet)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, v := range values {
		total += v
	}
	switch strings.ToUpper(c.name) {
	case "SUM":
		return total, nil
	case "AVERAGE":
		if len(values) == 0 {
			return 0, fmt.Errorf("formula: AVERAGE needs at least one value")
		}
		return total / float64(len(values)), nil
	default:
		return 0, fmt.Errorf("formula: unknown function %s", c.name)
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
		values = append(values, v)
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
