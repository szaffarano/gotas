// The logic for this package was taken from the original taskserver code
// https://github.com/GothenburgBitFactory/libshared/blob/1fa5dcbf53a280857e35436aef6beb6a37266e33/src/Pig.cpp

package task

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Pig is a parser helper taken from taskserver.
type Pig struct {
	value string
	idx   int
}

// NewPig creates a pig based on a string.
func NewPig(value string) *Pig {
	return &Pig{value: value}
}

// Skip moves the pig cursor until it finds the given rune and returns true,
// otherwise return false and the cursor is not changed.
func (p *Pig) Skip(ch rune) bool {
	if decoded, size := utf8.DecodeRuneInString(p.value[p.idx:]); decoded == ch {
		p.idx += size
		return true
	}

	return false
}

// GetUntil write the pig content to the writer until it finds a given rune
// (inclusive) and returns true, otherwise it returns false.
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

// GetQuoted removes the quote represented by the given rune and writes the
// result to the io.writer.  In case no text is quoted, returns false.
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
			isEscapedQuote := true
			for j >= start {
				ch, length := utf8.DecodeRuneInString(p.value[j:])
				if ch == utf8.RuneError || ch != '\\' {
					break
				}
				// Toggle flag for each further backslash encountered.
				isEscapedQuote = !isEscapedQuote
				j -= length
			}

			if isEscapedQuote {
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

// Eos returns true only when the end of stream was reached.
func (p *Pig) Eos() bool {
	ch, length := utf8.DecodeRuneInString(p.value[p.idx:])
	if ch == '\x00' || (ch == utf8.RuneError && length == 0) {
		return true
	}
	return false
}

// GetRemainder returns the remaining stream.
func (p *Pig) GetRemainder() string {
	ch, _ := utf8.DecodeRuneInString(p.value[p.idx:])
	if ch == utf8.RuneError {
		return ""
	}
	result := p.value[p.idx:]
	p.idx += len(result)

	return result
}

// SkipN move forward `n` runes and return true, in case it found an invalid rune or reach eos, returns false
func (p *Pig) SkipN(n int) bool {
	save := p.idx

	for count := 0; count < n; count++ {
		r, size := utf8.DecodeRuneInString(p.value[p.idx:])
		if r == utf8.RuneError {
			p.idx = save
			return false
		}
		p.idx += size
	}
	return true
}

// Cursor returns the current pig position.
func (p *Pig) Cursor() int {
	return p.idx
}

// RestoreTo change the pig index position or doesn't do anything if the
// requested index is out of range.  In any case returns the new (or
// unmodified) pig position.
func (p *Pig) RestoreTo(n int) int {
	if n > 0 && n < len(p.value) {
		p.idx = n
	}
	return p.idx
}

// GetDigit returns the next rune as an numeric value
func (p *Pig) GetDigit() (int, error) {
	return p.GetNDigits(1)
}

// GetDigit2 returns the next two runes as an numeric value
func (p *Pig) GetDigit2() (int, error) {
	return p.GetNDigits(2)
}

// GetDigit3 returns the next three runes as an numeric value
func (p *Pig) GetDigit3() (int, error) {
	return p.GetNDigits(3)
}

// GetDigit4 returns the next four runes as an numeric value
func (p *Pig) GetDigit4() (int, error) {
	return p.GetNDigits(4)
}

// GetValue  returns the remaining stream.
func (p *Pig) GetValue() string {
	return p.value[p.idx:]
}

// GetNDigits returns the next N runes as an numeric value
func (p *Pig) GetNDigits(n int) (int, error) {
	total := 0
	for i := 0; i < n; i++ {
		r, size := utf8.DecodeRuneInString(p.value[p.idx:])
		if r == utf8.RuneError || !unicode.IsDigit(r) {
			return 0, fmt.Errorf("no valid digit")
		}
		total += size
	}
	result, err := strconv.Atoi(p.value[p.idx : p.idx+total])
	if err != nil {
		return 0, err
	}
	p.idx += total
	return result, nil
}

// GetDigits returns the remaining runes as an numeric value
func (p *Pig) GetDigits() (int, error) {
	save := p.idx

	prev := p.idx
	for {
		r, size := utf8.DecodeRuneInString(p.value[p.idx:])
		if r == utf8.RuneError || !unicode.IsDigit(r) {
			p.idx = prev
			break
		}
		p.idx += size
		prev = p.idx
	}

	if p.idx > save {
		return strconv.Atoi(p.value[save : p.idx-save])
	}

	return 0, fmt.Errorf("no valid number found")
}

// Peek returns the current stream value as a rune but not modifies the index
// position.
func (p *Pig) Peek() rune {
	r, _ := utf8.DecodeRuneInString(p.value[p.idx:])
	return r
}

// Decode convert encoded "[" (&open;) and "]" (&close;) values to the proper rune.
func Decode(value string) string {
	if !strings.Contains(value, "&") {
		return value
	}

	value = strings.ReplaceAll(value, "&open;", "[")
	return strings.ReplaceAll(value, "&close;", "]")
}
