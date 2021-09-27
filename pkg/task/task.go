package task

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/apex/log"
	"github.com/google/uuid"
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

					task.data[name.String()] = parser.Decode(value.String())
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
		log.Debugf("record not recognized as format 4")
		return parseLegacy(raw)
	}

	// recalc_urgency = true;

	return task, nil
}

func parseLegacy(line string) (Task, error) {
	switch determineVersion(line) {
	// File format version 1, from 2006-11-27 - 2007-12-31, v0.x+ - v0.9.3
	case 1:
		return Task{}, fmt.Errorf("taskwarrior no longer supports file format 1, originally used between 27 November 2006 and 31 December 2007")

	// File format version 2, from 2008-1-1 - 2009-3-23, v0.9.3 - v1.5.0
	case 2:
		return Task{}, fmt.Errorf("taskwarrior no longer supports file format 2, originally used between 1 January 2008 and 12 April 2009")

	// File format version 3, from 2009-3-23 - 2009-05-16, v1.6.0 - v1.7.1
	case 3:
		return Task{}, fmt.Errorf("taskwarrior no longer supports file format 3, originally used between 23 March 2009 and 16 May 2009")

	// File format version 4, from 2009-05-16 - today, v1.7.1+
	case 4:
		break

	default:
		return Task{}, fmt.Errorf("unrecognized Taskwarrior file format or blank line in data")
	}

	// recalc_urgency = true
	// @TODO implement
	return Task{}, fmt.Errorf("not implemented")
}

func determineVersion(line string) int {
	// Version 2 looks like:
	//
	//   uuid status [tags] [attributes] description\n
	//
	// Where uuid looks like:
	//
	//   27755d92-c5e9-4c21-bd8e-c3dd9e6d3cf7
	//
	// Scan for the hyphens in the uuid, the following space, and a valid status
	// character.
	var validUuid bool
	var status byte
	if len(line) > 36 {
		_, err := uuid.Parse(line[0:36])
		status = line[37]
		validUuid = err == nil
	}

	if validUuid && (status == '-' || status == '+' || status == 'X' || status == 'r') {
		// Version 3 looks like:
		//
		//   uuid status [tags] [attributes] [annotations] description\n
		//
		// Scan for the number of [] pairs.
		tagAtts := strings.Index(line, "] [")
		attsAnno := strings.Index(string(line[tagAtts+1:]), "] [")
		annoDesc := strings.Index(string(line[attsAnno+1:]), "] ")
		if tagAtts != -1 && attsAnno != -1 && annoDesc != -1 {
			return 3
		} else {
			return 2
		}
	} else if line[0] == '[' && line[len(line)-1] == ']' && strings.Contains(line, `uuid:"`) {
		// Version 4 looks like:
		//
		//   [name:"value" ...]
		//
		// Scan for [, ] and :".
		return 4
	} else if strings.Contains(line, "X [") || (line[0] == '[' && line[len(line)-1] != ']' && len(line) > 3) {

		// Version 1 looks like:
		//
		//   [tags] [attributes] description\n
		//   X [tags] [attributes] description\n
		//
		// Scan for the first character being either the bracket or X.
		return 1
	}

	// Version 5?
	//
	// Fortunately, with the hindsight that will come with version 5, the
	// identifying characteristics of 1, 2, 3 and 4 may be modified such that if 5
	// has a UUID followed by a status, then there is still a way to differentiate
	// between 2, 3, 4 and 5.
	//
	// The danger is that a version 3 binary reads and misinterprets a version 4
	// file.  This is why it is a good idea to rely on an explicit version
	// declaration rather than chance positioning.

	// Zero means 'no idea'.
	return 0
}
