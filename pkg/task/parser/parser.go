package parser

import (
	"io"
	"strings"
	"unicode/utf8"
)

// The logic was taken from the original taskserver code
// https://github.com/GothenburgBitFactory/libshared/blob/1fa5dcbf53a280857e35436aef6beb6a37266e33/src/Pig.cpp
type Pig struct {
	value string
	idx   int
}

func NewPig(value string) *Pig {
	return &Pig{value: value}
}

func (p *Pig) Skip(ch rune) bool {
	if decoded, size := utf8.DecodeRuneInString(p.value[p.idx:]); decoded == ch {
		p.idx += size
		return true
	}

	return false
}

func (p *Pig) GetUntil(end rune, w io.Writer) bool {
	save := p.idx
	prev := p.idx

	for {
		ch, length := utf8.DecodeRuneInString(p.value[p.idx:])
		if ch == utf8.RuneError {
			break
		}
		p.idx += length

		if ch == end {
			p.idx = prev
			if _, err := w.Write([]byte(p.value[save:prev])); err != nil {
				return false
			}
			return true
		} else if p.Eos() {
			if _, err := w.Write([]byte(p.value[save:p.idx])); err != nil {
				return false
			}
			return true
		}
		prev = p.idx
	}

	return p.idx > save
}

func (p *Pig) GetQuoted(quote rune, w io.Writer) bool {
	ch, length := utf8.DecodeRuneInString(p.value[p.idx:])

	if ch == utf8.RuneError || ch != quote {
		return false
	}

	start := p.idx + length
	i := start

	for {
		k := strings.Index(p.value[i:], string(quote))
		if k == -1 {
			return false // Unclosed quote.  Short cut, not definitive.
		}
		i += k

		if i == start {
			// Empty quote
			p.idx += 2 * len(string(quote)) // Skip both quote chars
			if _, err := w.Write([]byte("")); err != nil {
				return false
			}
			return true
		}

		ch, length = utf8.DecodeRuneInString(p.value[i-1:])
		if ch == utf8.RuneError {
			break
		}

		if ch == '\\' {
			// Check for escaped backslashes.  Backtracking like this is not very
			// efficient, but is only done in extreme corner cases.

			j := i - (2 * len(string(quote))) // Start one character further left
			is_escaped_quote := true
			for j >= start {
				ch, length := utf8.DecodeRuneInString(p.value[j:])
				if ch == utf8.RuneError || ch != '\\' {
					break
				}
				// Toggle flag for each further backslash encountered.
				is_escaped_quote = !is_escaped_quote
				j -= length
			}

			if is_escaped_quote {
				i += length
				continue
			}
		}

		// None of the above applied, we must have found the closing quote char.
		if _, err := w.Write([]byte(p.value[start:i])); err != nil {
			return false
		}
		p.idx = i + len(string(quote)) // Skip closing quote char
		return true
	}

	// This should never be reached.  We could throw here instead.
	return false
}

func (p *Pig) Eos() bool {
	ch, length := utf8.DecodeRuneInString(p.value[p.idx:])
	if ch == '\x00' || (ch == utf8.RuneError && length == 0) {
		return true
	}
	return false
}

func (p *Pig) GetRemainder() string {
	ch, _ := utf8.DecodeRuneInString(p.value[p.idx:])
	if ch == utf8.RuneError {
		return ""
	}
	result := p.value[p.idx:]
	p.idx += len(result)

	return result
}

func Decode(value string) string {
	if !strings.Contains(value, "&") {
		return value
	}

	value = strings.ReplaceAll(value, "&open;", "[")
	return strings.ReplaceAll(value, "&close;", "]")
}
