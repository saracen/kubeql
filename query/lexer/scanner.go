package lexer

import (
	"unicode"
	"bufio"
	"io"
	"bytes"
	"strings"
	"strconv"
)

type TokenType int

const (
	Error TokenType = iota
	EOF
	Whitespace
	Ident

	Comma
	Arrow
	OpenParenthesis
	CloseParenthesis
	String
	Integer
	Float
	Dot

	And
	Or

	Add
	Subtract
	Multiply
	Divide
	True
	False

	Equal
	NotEqual
	LessThan
	LessThanEqual
	GreaterThan
	GreaterThanEqual

	Select
	From
	As
	Namespace
	Where

	JsonPath
	Jq
)

type Scanner struct {
	r *bufio.Reader

	idx, runeSize int
	buf  bytes.Buffer
	next struct {
		token  TokenType
		text   string
		offset int
	}
}

func NewScanner(r io.Reader) *Scanner {
	s := &Scanner{r: bufio.NewReader(r)}
	s.Scan()

	return s
}

func (s *Scanner) Scan() (TokenType, int, string) {
	token, offset, text := s.next.token, s.next.offset, s.next.text

	s.next.token, s.next.offset, s.next.text = s.scan(), s.idx, s.buf.String()

	return token, offset, text
}


func (s *Scanner) Peek() TokenType {
	return s.next.token
}

func (s *Scanner) scan() TokenType {
	s.buf.Reset()
	r := s.read()

	switch {
	case unicode.IsSpace(r):
		s.unread()
		s.scanWhitespace()
		// skip whitespace
		return s.scan()

	case isIdentifier(r):
		s.unread()
		return s.scanIdent()

	case r == '"' || r == '\'' || r == '`':
		s.unread()
		return s.scanString()

	case unicode.IsDigit(r) || (r == '.' && unicode.IsDigit(s.peek())):
		s.unread()
		return s.scanNumber()
	}

	s.buf.WriteRune(r)

	switch r {
	case '.':
		return Dot

	case '-':
		if s.peek() == '>' {
			s.buf.WriteRune(s.read())
			return Arrow
		}
		return Subtract

	case '+':
		return Add

	case '*':
		return Multiply

	case '/':
		return Divide

	case '=':
		return Equal

	case '!':
		if s.peek() == '=' {
			s.buf.WriteRune(s.read())
			return NotEqual
		}
		return Error

	case '<':
		if s.peek() == '=' {
			s.buf.WriteRune(s.read())
			return LessThanEqual
		}
		return LessThan

	case '>':
		if s.peek() == '=' {
			s.buf.WriteRune(s.read())
			return GreaterThanEqual
		}
		return GreaterThan

	case ',':
		return Comma

	case '(':
		return OpenParenthesis

	case ')':
		return CloseParenthesis

	case 0:
		return EOF
	}

	return TokenType(r)
}

func (s *Scanner) scanWhitespace() TokenType {
	for unicode.IsSpace(s.read()) {
		// absorb
	}
	s.unread()

	return Whitespace
}

func (s *Scanner) scanIdent() TokenType {
	for {
		r := s.peek()
		if !isIdentifier(r) && !unicode.IsDigit(r) {
			break
		}

		s.buf.WriteRune(s.read())
	}

	ident := strings.ToLower(s.buf.String())
	switch ident {
	case "or":
		return Or
	case "and":
		return And
	case "select":
		return Select
	case "from":
		return From
	case "as":
		return As
	case "namespace":
		return Namespace
	case "where":
		return Where
	case "true":
		return True
	case "false":
		return False
	case "jsonpath":
		return JsonPath
	case "jq":
		return Jq
	}

	return Ident
}

func (s *Scanner) scanString() TokenType {
	open := s.read()

	for {
		r := s.read()
		if r == rune(0) {
			return Error
		}

		if r == open {
			break
		}

		if r == '\\' {
			p := s.peek()
			if p == '\'' || p == '"' || p == '`' {
				s.buf.WriteRune(s.read())
				continue
			}
		}

		s.buf.WriteRune(r)
	}

	return String
}

func (s *Scanner) scanNumber() TokenType {
	for {
		r := s.peek()
		if r != '.' && !unicode.IsDigit(r) {
			break
		}
		s.buf.WriteRune(s.read())
	}

	if _, err := strconv.Atoi(s.buf.String()); err == nil {
		return Integer
	}
	if _, err := strconv.ParseFloat(s.buf.String(), 64); err == nil {
		return Float
	}

	return Error
}

func (s *Scanner) read() rune {
	ch, i, err := s.r.ReadRune()
	if err != nil {
		return 0
	}

	s.runeSize = i
	s.idx += i

	return ch
}

func (s *Scanner) unread() {
	s.r.UnreadRune()
	s.idx -= s.runeSize
	s.runeSize = 0
}

func (s *Scanner) peek() rune {
	r := s.read()
	s.unread()

	return r
}

func isIdentifier(r rune) bool {
  	return r == '_' || unicode.IsLetter(r)
}
