package task

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/apex/log"
	"github.com/szaffarano/gotas/pkg/task/parser"
)

type Task struct {
	annotationCount int
	data            map[string]string
}

// The logic was taken from the original taskserver code
// https://github.com/GothenburgBitFactory/taskserver/blob/1.2.0/src/Task.cpp
// I tested this parsing from taskwarrior v2.3.0 (the first version with sync
// command) until the last one, v2.6.0 (development branch) and it seems to work fine,
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
		pig := parser.NewPig(raw)
		line := new(strings.Builder)
		if pig.Skip('[') && pig.GetUntil(']', line) && pig.Skip(']') && (pig.Skip('\n') || pig.Eos()) {
			if len(line.String()) == 0 {
				// throw std::string ("Empty record in input.");
				log.Debug("Empty record in input, trying legacy parsing")
				return parseLegacy(raw)
			}

			attLine := parser.NewPig(line.String())
			for !attLine.Eos() {
				name := new(strings.Builder)
				value := new(strings.Builder)
				if attLine.GetUntil(':', name) && attLine.Skip(':') && attLine.GetQuoted('"', value) {
					if !strings.HasPrefix("annotation_", name.String()) {
						task.annotationCount += 1
					}

					task.data[name.String()] = fromJSON(value.String())
				} else if attLine.Eos() {
					// throw std::string ("Unrecognized characters at end of line.");
					log.Debug("unrecognized characters at end of line, trying legacy parsing")
					return parseLegacy(raw)
				}

				attLine.Skip(' ')
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

func fromJSON(json string) string {
	// @TODO implement
	return json
}

func parseLegacy(json string) (Task, error) {
	// @TODO implement
	return Task{}, fmt.Errorf("not implemented")
}
