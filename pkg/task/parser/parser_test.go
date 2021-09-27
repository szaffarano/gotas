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

func TestGetQuoted(t *testing.T) {
	cases := []struct {
		title    string
		value    string
		quote    rune
		expected string
		success  bool
	}{
		{"GetQuoted for simple string", `"foobar"`, '"', "foobar", true},
		{"GetQuoted for for unquoted string", `foobar`, '"', "", false},
		{"GetQuoted for for empty quoted string", `""`, '"', "", true},
		{"GetQuoted for for unbalanced quoted string", `"foo`, '"', "", false},
		{"GetQuoted for double escaped string", `"foo\\"bar`, '"', `foo\\`, true},
		{"GetQuoted for multiple escaped", "\"one\\\\\"", '"', "one\\\\", true},
		{"GetQuoted for escaped string", `"foo\"bar"`, '"', `foo\"bar`, true},
		{"GetQuoted for double escaped string", `"foo\a\b\"bar"`, '"', `foo\a\b\"bar`, true},
		{"GetQuoted with alternative UTF-8 rune", `日foobar日`, '日', `foobar`, true},
		{"GetQuoted with alternative UTF-8 rune and escaped", `日foo\日bar日`, '日', `foo\日bar`, true},
		{"GetQuoted with invalid UTF-8 rune should fail", "'foobar\xbd\xb2'", '\'', "", false},
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
		{"GetRemainder works in the middle of a string", "123", '1', "23"},
		{"GetRemainder works from the beginning of a string", "123", 0, "123"},
		{"GetRemainder fails with invalid UTF-8 string", "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98", 0, ""},
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
		{"not eos in the beginning of a string", "123", 0, false},
		{"eos at the end of a string", "1", '1', true},
		{"not eos in invalid UTF-8 string", "\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98", 0, false},
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
		{"skip until inexistent rune", "123", '4', "123", true},
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

	cases = []struct {
		title    string
		value    string
		until    rune
		expected string
		success  bool
	}{
		{"fails with invalid writer", "hello world", ' ', "", false},
		{"fails with invalid writer at the end", "hello world", 'x', "", false},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			p := NewPig("hello world")
			w := new(piggyWriter)

			result := p.GetUntil(c.until, w)

			assert.False(t, result)
		})
	}

	t.Run("successive GetUntil requests", func(t *testing.T) {
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

		p.Skip(':')

		w = new(strings.Builder)
		result = p.GetUntil(':', w)
		assert.True(t, result)
		assert.Equal(t, "ghi", w.String())

		p.Skip(':')

		w = new(strings.Builder)
		result = p.GetUntil(':', w)
		assert.False(t, result)
	})
}

func TestJsonDecode(t *testing.T) {
	cases := []struct {
		value    string
		expected string
	}{
		{`1\"2`, "1\\\"2"},
		{`1\b2`, "1\\b2"},
		{`1\f2`, "1\\f2"},
		{`1\n2`, "1\\n2"},
		{`1\r2`, "1\\r2"},
		{`1\t2`, "1\\t2"},
		{`1\\2`, "1\\\\2"},
		{`one\\`, "one\\\\"},
		{"1\x02", "1\x02"},
		{"1€2", "1\u20ac2"},
		{"&open;hello&close;", "[hello]"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("decoding %v", c.value), func(t *testing.T) {
			assert.Equal(t, c.expected, Decode(c.value))
		})
	}

}
