package formula

import "fmt"

type node interface{}

type numberNode struct{ val float64 }
type cellNode struct{ ref string }
type binaryNode struct {
	op          tokenKind
	left, right node
}
type unaryNode struct {
	op      tokenKind
	operand node
}
type callNode struct {
	name string
	args []node
}
type rangeNode struct {
	from, to string
}

type parser struct {
	lex *lexer
	cur token
}

func newParser(src string) (*parser, error) {
	p := &parser{lex: newLexer(src)}
	return p, p.advance()
}

func (p *parser) advance() error {
	t, err := p.lex.next()
	if err != nil {
		return err
	}
	p.cur = t
	return nil
}

var precedence = map[tokenKind]int{tokPlus: 1, tokMinus: 1, tokStar: 2, tokSlash: 2}

func parseExpr(src string) (node, error) {
	p, err := newParser(src)
	if err != nil {
		return nil, err
	}
	n, err := p.parseBinary(1)
	if err != nil {
		return nil, err
	}
	if p.cur.kind != tokEOF {
		return nil, fmt.Errorf("formula: unexpected token %q", p.cur.text)
	}
	return n, nil
}

func (p *parser) parseBinary(minPrec int) (node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		prec, ok := precedence[p.cur.kind]
		if !ok || prec < minPrec {
			return left, nil
		}
		op := p.cur.kind
		if err := p.advance(); err != nil {
			return nil, err
		}
		right, err := p.parseBinary(prec + 1)
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: op, left: left, right: right}
	}
}

func (p *parser) parseUnary() (node, error) {
	if p.cur.kind == tokMinus || p.cur.kind == tokPlus {
		op := p.cur.kind
		if err := p.advance(); err != nil {
			return nil, err
		}
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{op: op, operand: operand}, nil
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (node, error) {
	switch p.cur.kind {
	case tokNumber:
		n := numberNode{val: p.cur.num}
		return n, p.advance()
	case tokLParen:
		if err := p.advance(); err != nil {
			return nil, err
		}
		n, err := p.parseBinary(1)
		if err != nil {
			return nil, err
		}
		if p.cur.kind != tokRParen {
			return nil, fmt.Errorf("formula: expected )")
		}
		return n, p.advance()
	case tokIdent:
		return p.parseIdent()
	default:
		return nil, fmt.Errorf("formula: unexpected token %q", p.cur.text)
	}
}

func (p *parser) parseIdent() (node, error) {
	name := p.cur.text
	if err := p.advance(); err != nil {
		return nil, err
	}
	if p.cur.kind == tokLParen {
		return p.parseCall(name)
	}
	if !looksLikeRef(name) {
		return nil, fmt.Errorf("formula: %q is not a valid cell reference", name)
	}
	if p.cur.kind == tokColon {
		if err := p.advance(); err != nil {
			return nil, err
		}
		if p.cur.kind != tokIdent {
			return nil, fmt.Errorf("formula: expected cell reference after :")
		}
		to := p.cur.text
		if !looksLikeRef(to) {
			return nil, fmt.Errorf("formula: %q is not a valid cell reference", to)
		}
		if err := p.advance(); err != nil {
			return nil, err
		}
		return rangeNode{from: name, to: to}, nil
	}
	return cellNode{ref: name}, nil
}

func (p *parser) parseCall(name string) (node, error) {
	if err := p.advance(); err != nil {
		return nil, err
	}
	var args []node
	if p.cur.kind != tokRParen {
		for {
			arg, err := p.parseBinary(1)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if p.cur.kind != tokComma {
				break
			}
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
	}
	if p.cur.kind != tokRParen {
		return nil, fmt.Errorf("formula: expected ) after arguments to %s", name)
	}
	return callNode{name: name, args: args}, p.advance()
}
