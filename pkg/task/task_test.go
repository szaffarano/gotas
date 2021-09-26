package task

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
			p := Pig{value: c.value}
			w := new(strings.Builder)

			worked := p.getQuoted(c.quote, w)

			if !c.success {
				assert.False(t, worked)
			} else {
				assert.True(t, worked)
				assert.Equal(t, c.expected, w.String())
			}
		})
	}

	t.Run("fail with invalid writer", func(t *testing.T) {
		p := Pig{value: `""`}
		w := new(piggyWriter)

		worked := p.getQuoted('"', w)
		assert.False(t, worked)
	})

	t.Run("fail with invalid writer", func(t *testing.T) {
		p := Pig{value: `"hello"`}
		w := new(piggyWriter)

		worked := p.getQuoted('"', w)
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
			p := Pig{value: c.value}

			if c.skip != 0 {
				result := p.skip(c.skip)
				assert.True(t, result)
			}

			actual := p.getRemainder()

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
			p := Pig{value: c.value}

			result := p.skip(c.skip)
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
			p := Pig{value: c.value}
			// w := new(strings.Builder)

			if c.skip != 0 {
				result := p.skip(c.skip)
				assert.True(t, result)
			}

			assert.Equal(t, c.expected, p.eos())
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
			p := Pig{value: c.value}
			w := new(strings.Builder)

			result := p.getUntil(c.until, w)

			assert.Equal(t, c.success, result)
			assert.Equal(t, c.expected, w.String())
		})
	}

	t.Run("fails with invalid writer", func(t *testing.T) {
		p := Pig{value: "hello world"}
		w := new(piggyWriter)

		result := p.getUntil(' ', w)

		assert.False(t, result)
	})

	t.Run("fails with invalid writer at the end", func(t *testing.T) {
		p := Pig{value: "hello world"}
		w := new(piggyWriter)

		result := p.getUntil('x', w)

		assert.False(t, result)
	})

	t.Run("new case", func(t *testing.T) {
		p := Pig{value: "abc:def:ghi"}
		w := new(strings.Builder)

		result := p.getUntil(':', w)

		assert.True(t, result)
		assert.Equal(t, "abc", w.String())

		p.skip(':')

		w = new(strings.Builder)
		result = p.getUntil(':', w)
		assert.True(t, result)
		assert.Equal(t, "def", w.String())

	})
}

func TestNewTask(t *testing.T) {
	cases := []struct {
		title   string
		raw     string
		success bool
		values  map[string]string
	}{
		{
			"simple task works",
			`[description:"Some task" entry:"123" status:"pending" uuid:"456"]`,
			true,
			map[string]string{
				"description": "Some task",
				"uuid":        "456",
				"status":      "pending",
				"entry":       "123",
			},
		},
		{
			"additional characters at the end of the task fails",
			`[description:"Some task" entry:"123" status:"pending" uuid:"456a" abc def]`,
			false,
			nil,
		},
		{
			"empty task fails",
			`[]`,
			false,
			nil,
		},
		{
			"json format fails (not implemented)",
			`{"description":"Test 2","end":"20210925T160632Z","entry":"20210925T160542Z","modified":"20210925T160632Z","status":"completed","uuid":"123"}`,
			false,
			nil,
		},
		{
			"empty string fails",
			"",
			false,
			nil,
		},
		{
			"string with invalid rune fails",
			"\xbd\xb2",
			false,
			nil,
		},
		{
			"format FF3 fails",
			`a2b5f6fc-7285-75cc-90b9-abf624a8457e - [] [entry:1632687645 priority: project:] [] Some task`,
			false,
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			task, err := NewTask(c.raw)

			a := assert.New(t)
			if c.success {
				a.Nil(err)
				a.NotNil(task.data)
				a.Equal(c.values, task.data)
			} else {
				a.NotNil(err)
			}
		})
	}
}
