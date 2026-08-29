package formula

import (
	"fmt"
	"strconv"
)

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokNumber
	tokIdent
	tokPlus
	tokMinus
	tokStar
	tokSlash
	tokLParen
	tokRParen
	tokComma
)

type token struct {
	kind tokenKind
	text string
	num  float64
}

type lexer struct {
	src string
	pos int
}

func newLexer(src string) *lexer {
	return &lexer{src: src}
}

func (l *lexer) next() (token, error) {
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t') {
		l.pos++
	}
	if l.pos >= len(l.src) {
		return token{kind: tokEOF}, nil
	}
	c := l.src[l.pos]
	single := map[byte]tokenKind{
		'+': tokPlus, '-': tokMinus, '*': tokStar, '/': tokSlash,
		'(': tokLParen, ')': tokRParen, ',': tokComma,
	}
	if kind, ok := single[c]; ok {
		l.pos++
		return token{kind: kind, text: string(c)}, nil
	}
	switch {
	case isDigit(c) || c == '.':
		start := l.pos
		for l.pos < len(l.src) && (isDigit(l.src[l.pos]) || l.src[l.pos] == '.') {
			l.pos++
		}
		text := l.src[start:l.pos]
		n, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return token{}, fmt.Errorf("formula: bad number %q", text)
		}
		return token{kind: tokNumber, text: text, num: n}, nil
	case isLetter(c):
		start := l.pos
		for l.pos < len(l.src) && (isLetter(l.src[l.pos]) || isDigit(l.src[l.pos])) {
			l.pos++
		}
		return token{kind: tokIdent, text: l.src[start:l.pos]}, nil
	default:
		return token{}, fmt.Errorf("formula: unexpected character %q", c)
	}
}

func isDigit(c byte) bool  { return c >= '0' && c <= '9' }
func isLetter(c byte) bool { return c >= 'A' && c <= 'Z' }
