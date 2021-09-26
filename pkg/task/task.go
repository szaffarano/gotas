package task

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/apex/log"
)

type Task struct {
	annotationCount int
	data            map[string]string
}

// Logic taken from original taskserver code
// https://github.com/GothenburgBitFactory/taskserver/blob/1.2.0/src/Task.cpp
// I tested this parsing from taskwarrior v2.3.0 (the first version with sync
// command) until last 2.6.0 (development branch) and it seems to work fine,
// always receiving JSON payloads
func NewTask(raw string) (Task, error) {
	task := Task{
		data:            make(map[string]string),
		annotationCount: 0,
	}

	rune, size := utf8.DecodeRuneInString(raw)
	switch rune {
	// first try, format v4
	case '[':
		pig := Pig{value: raw}
		line := new(strings.Builder)
		if pig.skip('[') && pig.getUntil(']', line) && pig.skip(']') && (pig.skip('\n') || pig.eos()) {
			if len(line.String()) == 0 {
				// throw std::string ("Empty record in input.");
				log.Debug("Empty record in input, trying legacy parsing")
				return parseLegacy(raw)
			}

			attLine := Pig{value: line.String()}
			for !attLine.eos() {
				name := new(strings.Builder)
				value := new(strings.Builder)
				if attLine.getUntil(':', name) && attLine.skip(':') && attLine.getQuoted('"', value) {

					if !strings.HasPrefix("annotation_", name.String()) {
						task.annotationCount += 1
					}

					task.data[name.String()] = fromJSON(value.String())
				} else if attLine.eos() {
					// throw std::string ("Unrecognized characters at end of line.");
					log.Debug("unrecognized characters at end of line, trying legacy parsing")
					return parseLegacy(raw)
				}

				attLine.skip(' ')
			}
		}
	case '{':
		// parseJSON (input);
		// @TODO implement json parsing
		return Task{}, fmt.Errorf("json format not implemented")
	case utf8.RuneError:
		if size == 0 {
			return Task{}, fmt.Errorf("empty string")
		} else {
			return Task{}, fmt.Errorf("invalid string")
		}
	default:
		// throw std::string ("Record not recognized as format 4.");
		// @TODO parseLegacy
		return Task{}, fmt.Errorf("record not recognized as format 4")
	}

	// recalc_urgency = true;

	return task, nil
}

type Pig struct {
	value string
	idx   int
}

func (p *Pig) skip(ch rune) bool {
	if decoded, size := utf8.DecodeRuneInString(p.value[p.idx:]); decoded == ch {
		p.idx += size
		return true
	}

	return false
}

func (p *Pig) getUntil(end rune, w io.Writer) bool {
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
		} else if p.eos() {
			if _, err := w.Write([]byte(p.value[save:p.idx])); err != nil {
				return false
			}
			return true
		}
		prev = p.idx
	}

	return p.idx > save
}

func (p *Pig) getQuoted(quote rune, w io.Writer) bool {
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

func (p *Pig) eos() bool {
	ch, length := utf8.DecodeRuneInString(p.value[p.idx:])
	if ch == '\x00' || (ch == utf8.RuneError && length == 0) {
		return true
	}
	return false
}

func (p *Pig) getRemainder() string {
	ch, _ := utf8.DecodeRuneInString(p.value[p.idx:])
	if ch == utf8.RuneError {
		return ""
	}
	result := p.value[p.idx:]
	p.idx += len(result)

	return result
}

func fromJSON(json string) string {
	// @TODO implement
	return json
}

func parseLegacy(json string) (Task, error) {
	// @TODO implement
	return Task{}, fmt.Errorf("not implemented")
}
