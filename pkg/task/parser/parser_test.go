package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type piggyWriter string

func (c piggyWriter) Write(buf []byte) (int, error) {
	return 0, fmt.Errorf("bum!")
}

// Gets quote content:      "foobar" -> foobar      (for c = '"')
// Handles escaped quotes:  "foo\"bar" -> foo\"bar  (for c = '"')
func TestGetQuoted(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		quote    rune
		expected string
		success  bool
	}{
		{"get quoted for simple string", `"foobar"`, '"', "foobar", true},
		{"get quoted for for unquoted string", `foobar`, '"', "", false},
		{"get quoted for for empty quoted string", `""`, '"', "", true},
		{"get quoted for for unbalanced quoted string", `"foo`, '"', "", false},
		{"get quoted for double escaped string", `"foo\\"bar`, '"', `foo\\`, true},
		{"get quoted for multiple escaped", "\"one\\\\\"", '"', "one\\\\", true},
		{"get quoted for escaped string", `"foo\"bar"`, '"', `foo\"bar`, true},
		{"get quoted for double escaped string", `"foo\a\b\"bar"`, '"', `foo\a\b\"bar`, true},
		{"get quoted with alternative utf8 rune", `日foobar日`, '日', `foobar`, true},
		{"get quoted with alternative utf8 rune and escaped", `日foo\日bar日`, '日', `foo\日bar`, true},
		{"get quoted with invalid utf8 rune should fail", "'foobar\xbd\xb2'", '\'', "", false},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewPig(c.value)
			w := new(strings.Builder)

			worked := p.GetQuoted(c.quote, w)

			if !c.success {
				assert.False(t, worked)
			} else {
				assert.True(t, worked)
				assert.Equal(t, c.expected, w.String())
			}
		})
	}

	t.Run("fail with invalid writer", func(t *testing.T) {
		p := NewPig(`""`)
		w := new(piggyWriter)

		worked := p.GetQuoted('"', w)
		assert.False(t, worked)
	})

	t.Run("fail with invalid writer", func(t *testing.T) {
		p := NewPig(`"hello"`)
		w := new(piggyWriter)

		worked := p.GetQuoted('"', w)
		assert.False(t, worked)
	})
}

func TestGetRemainder(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		skip     rune
		expected string
	}{
		{"reminder works in the middle of a string", "123", '1', "23"},
		{"reminder works form begin of a string", "123", 0, "123"},
		{"reminder fails with unvalid utf8 string", "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98", 0, ""},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewPig(c.value)

			if c.skip != 0 {
				result := p.Skip(c.skip)
				assert.True(t, result)
			}

			actual := p.GetRemainder()

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestSkip(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		skip     rune
		expected bool
	}{
		{"skip until valid rune", "123", '1', true},
		{"skip until invalid rune", "123", '2', false},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewPig(c.value)

			result := p.Skip(c.skip)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestEos(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		skip     rune
		expected bool
	}{
		{"not eos in the middle of a string", "123", '1', false},
		{"not eos in the begining of a string", "123", 0, false},
		{"eos at the end of a string", "1", '1', true},
		{"not eos in invalid utf8 string", "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98", 0, false},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewPig(c.value)
			// w := new(strings.Builder)

			if c.skip != 0 {
				result := p.Skip(c.skip)
				assert.True(t, result)
			}

			assert.Equal(t, c.expected, p.Eos())
		})
	}
}

func TestGetUntil(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		until    rune
		expected string
		success  bool
	}{
		{"skip until de middle of the string", "123", '2', "1", true},
		{"skip until unexistent rune", "123", '4', "123", true},
		{"skip duplicated rune", "hello world .", ' ', "hello", true},
		{"fails with invalid rune at the beginning", "\xbd\xb2hello world", ' ', "", false},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewPig(c.value)
			w := new(strings.Builder)

			result := p.GetUntil(c.until, w)

			assert.Equal(t, c.success, result)
			assert.Equal(t, c.expected, w.String())
		})
	}

	t.Run("fails with invalid writer", func(t *testing.T) {
		p := NewPig("hello world")
		w := new(piggyWriter)

		result := p.GetUntil(' ', w)

		assert.False(t, result)
	})

	t.Run("fails with invalid writer at the end", func(t *testing.T) {
		p := NewPig("hello world")
		w := new(piggyWriter)

		result := p.GetUntil('x', w)

		assert.False(t, result)
	})

	t.Run("new case", func(t *testing.T) {
		p := NewPig("abc:def:ghi")
		w := new(strings.Builder)

		result := p.GetUntil(':', w)

		assert.True(t, result)
		assert.Equal(t, "abc", w.String())

		p.Skip(':')

		w = new(strings.Builder)
		result = p.GetUntil(':', w)
		assert.True(t, result)
		assert.Equal(t, "def", w.String())

	})
}
