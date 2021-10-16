// Package task defines the common model used by taskd
// in particular the Task type as well some constants and definitions
package task

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/szaffarano/gotas/pkg/logger"
	"github.com/szaffarano/gotas/pkg/parser"
)

const (
	// DateLayout is the format used to represent dates. The original
	// taskserver implementation allows many different formats but AFAIK those
	// are for the client-side (task warrior).  At least 2.3.0+ task warrior
	// clients, always send dates in this format.
	DateLayout = "20060102T150405Z"
)

// Constants associated to configuration entries.
const (
	Confirmation = "confirmation"
	Extensions   = "extensions"
	IPLog        = "ip.log"
	Log          = "log"
	PidFile      = "pid.file"
	QueueSize    = "queue.size"
	RequestLimit = "request.limit"
	Root         = "root"
	BindAddress  = "server"
	Trust        = "trust"
	Verbose      = "verbose"
	ClientCert   = "client.cert"
	ClientKey    = "client.key"
	ServerKey    = "server.key"
	ServerCert   = "server.cert"
	ServerCrl    = "server.crl"
	CaCert       = "ca.cert"
)

var (
	attributeTypes = map[string]string{
		"depends":      "string",
		"description":  "string",
		"due":          "date",
		"end":          "date",
		"entry":        "date",
		"id":           "string",
		"imask":        "numeric",
		"mask":         "string",
		"modification": "date",
		"modified":     "date",
		"parent":       "string",
		"priority":     "string",
		"project":      "string",
		"recur":        "duration",
		"scheduled":    "date",
		"start":        "date",
		"status":       "string",
		"tags":         "string",
		"until":        "date",
		"urgency":      "string",
		"uuid":         "string",
		"wait":         "date",
	}

	// ErrorCodes are the error codes and status descriptions sent back to the
	// user.
	ErrorCodes = map[int]string{
		// 2xx Success.
		200: "Ok",
		201: "No change",
		202: "Decline",

		// 3xx Partial success.
		300: "Deprecated request type",
		301: "Redirect",
		302: "Retry",

		// 4xx Client error.
		// "401": "Failure",
		400: "Malformed data",
		401: "Unsupported encoding",
		420: "Server temporarily unavailable",
		430: "Access denied",
		431: "Account suspended",
		432: "Account terminated",

		// 5xx Server error.
		500: "Syntax error in request",
		501: "Syntax error, illegal parameters",
		502: "Not implemented",
		503: "Command parameter not implemented",
		504: "Request too big",
	}
)

var log *logger.Logger

func init() {
	log = logger.Log()
}

// Task represents each task sent by the client to be synced
type Task struct {
	annotationCount int
	data            map[string]string
}

// NewTask parses a raw string as a taskwarrior Task.
//
// The parsing algorithm was taken from the original taskserver code
// https://github.com/GothenburgBitFactory/taskserver/blob/1.2.0/src/Task.cpp
//
// I tested this parser using taskwarrior payloads from v2.3.0 (the first sync
// command implementation) until the last one, v2.6.0 (development branch) and
// it seems to work fine, always receiving JSON payloads.
func NewTask(raw string) (Task, error) {
	rune, _ := utf8.DecodeRuneInString(raw)
	switch rune {
	// first try, format v4
	case '[':
		return parseV4(raw)
	case '{':
		return parseJSON(raw)
	case utf8.RuneError:
		return Task{}, fmt.Errorf("invalid string")
	default:
		log.Debugf("record not recognized as format 4")
		return parseLegacy(raw)
	}
}

func parseV4(raw string) (Task, error) {
	task := Task{
		data:            make(map[string]string),
		annotationCount: 0,
	}

	pig := parser.NewPig(raw)
	line := new(strings.Builder)

	if pig.Skip('[') && pig.GetUntil(']', line) && pig.Skip(']') && (pig.Skip('\n') || pig.Eos()) {
		if len(line.String()) == 0 {
			log.Debug("Empty record in input, trying legacy parsing")
			return parseLegacy(raw)
		}

		attLine := parser.NewPig(line.String())
		for !attLine.Eos() {
			name := new(strings.Builder)
			value := new(strings.Builder)
			if attLine.GetUntil(':', name) && attLine.Skip(':') && attLine.GetQuoted('"', value) {
				if !strings.HasPrefix(name.String(), "annotation_") {
					task.annotationCount++
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

func parseJSON(line string) (Task, error) {
	lineAsJSON := make(map[string]interface{})

	if err := json.Unmarshal([]byte(line), &lineAsJSON); err != nil {
		return Task{}, fmt.Errorf("parsing json: %v", err.Error())
	}

	uuid := fmt.Sprintf("%v", lineAsJSON["uuid"])
	t := Task{
		data: map[string]string{
			"uuid": uuid,
		},
	}

	for attrName, attrValue := range lineAsJSON {
		// If the attribute is a recognized column.
		if attrType := attributeTypes[attrName]; attrType != "" {
			if attrName == "id" {
				// Any specified id is ignored.
				continue
			} else if attrName == "urgency" {
				// Urgency, if present, is ignored.
				continue
			} else if attrName == "modification" {
				// TW-1274 Standardization.
				ts, err := time.Parse(DateLayout, fmt.Sprintf("%v", attrValue))
				if err != nil {
					return Task{}, fmt.Errorf("parsing date in %v field, %v: %v", attrName, attrValue, err.Error())
				}
				t.data["modified"] = fmt.Sprintf("%d", ts.UTC().Unix())
			} else if attrType == "date" {
				// Dates are converted from ISO to epoch.
				ts, err := time.Parse(DateLayout, fmt.Sprintf("%v", attrValue))
				if err != nil {
					return Task{}, fmt.Errorf("parsing date in %v field, %v: %v", attrName, attrValue, err.Error())
				}
				t.data[attrName] = fmt.Sprintf("%d", ts.UTC().Unix())
			} else if attrName == "tags" {
				tags, err := parseTags(attrValue)
				if err != nil {
					return Task{}, err
				}
				for _, tag := range tags {
					t.addTag(tag)
				}
			} else if attrName == "depends" {
				dependencies, err := parseDepends(attrValue)
				if err != nil {
					return Task{}, err
				}

				for _, dep := range dependencies {
					if err := t.addDependency(dep); err != nil {
						return Task{}, err
					}
				}
			} else {
				// Other types are simply added.
				// json.Unmarshal already decoded the `\uxxxx` escaped unicode
				t.data[attrName] = fmt.Sprintf("%v", attrValue)
			}
		} else {
			// UDA orphans and annotations do not have columns.

			if attrName == "annotations" {
				entries, err := parseAnnoations(attrValue)
				if err != nil {
					return Task{}, err
				}

				for _, e := range entries {
					t.data[e[0]] = e[1]
				}
			} else { // UDA Orphan - must be preserved.
				t.data[attrName] = fmt.Sprintf("%v", attrValue)
			}
		}
	}
	return t, nil
}

func parseTags(attrValue interface{}) ([]string, error) {
	var tags []string
	switch value := attrValue.(type) {
	case []interface{}:
		// Tags are an array of JSON strings.
		for _, tag := range value {
			tags = append(tags, fmt.Sprintf("%v", tag))
		}
	case string:
		// This is a temporary measure to accommodate a malformed JSON message
		// from Mirakel sync.
		// 2016-02-21 Mirakel dropped sync support in late 2015. This can be
		//            removed in a later release.
		tags = append(tags, value)
	default:
		return nil, fmt.Errorf("invalid type for field tags: %v", attrValue)
	}
	return tags, nil
}

func parseDepends(attrValue interface{}) ([]string, error) {
	var deps []string
	switch value := attrValue.(type) {
	case []interface{}:
		// Dependencies can be exported as an array of strings.
		// 2016-02-21: This will be the only option in future releases.
		//             See other 2016-02-21 comments for details.
		for _, dependency := range value {
			deps = append(deps, fmt.Sprintf("%v", dependency))
		}
	case string:
		// Dependencies can be exported as a single comma-separated string.
		// 2016-02-21: Deprecated - see other 2016-02-21 comments for details.
		for _, dependency := range strings.Split(value, ",") {
			deps = append(deps, fmt.Sprintf("%v", dependency))
		}
	default:
		return nil, fmt.Errorf("depends type not match: %v", value)
	}
	return deps, nil
}

func parseAnnoations(attrValue interface{}) ([][]string, error) {
	// Annotations are an array of JSON objects with 'entry' and
	// 'description' values and must be converted.
	var entries [][]string
	if annotations, ok := attrValue.([]interface{}); ok {
		for _, item := range annotations {
			if annotation, ok := item.(map[string]interface{}); ok {
				entry := make([]string, 2)

				when, ok := annotation["entry"]
				if !ok {
					return nil, fmt.Errorf("annotation is missing an entry date: %v", annotation)
				}
				what, ok := annotation["description"]
				if !ok {
					return nil, fmt.Errorf("annotation is missing a description: %v", annotation)
				}

				ts, err := time.Parse(DateLayout, fmt.Sprintf("%v", when))
				if err != nil {
					return nil, fmt.Errorf("invalid date format %q: %v", when, err.Error())
				}
				name := fmt.Sprintf("annotation_%v", ts.UTC().Unix())

				entry[0] = name
				entry[1] = fmt.Sprintf("%v", what)
				entries = append(entries, entry)
			} else {
				return nil, fmt.Errorf("annotations type inside list does not match: %T", attrValue)
			}
		}
		return entries, nil
	}
	return nil, fmt.Errorf("annotations type does not match: %T", attrValue)
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
	var validUUID bool
	var status byte
	if len(line) > 36 {
		_, err := uuid.Parse(line[0:36])
		status = line[37]
		validUUID = err == nil
	}

	if validUUID && (status == '-' || status == '+' || status == 'X' || status == 'r') {
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
		}
		return 2
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

// Get returns the given task attribute or the zero value if it doesn't exists.
func (t *Task) Get(name string) string {
	return t.data[name]
}

// Set sets or overrides the given attribute to the task.
func (t *Task) Set(name, value string) {
	t.data[name] = value
}

// GetInt returns the given task attribute as an integer or the zero value if it
// doesn't exists or it can't be parsed as integer.
func (t *Task) GetInt(name string) int {
	if value, ok := t.data[name]; ok {
		num, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

// GetDate returns the given task attribute as an UTC date or the zero value if it
// doesn't exists or it can't be parsed as a date.
func (t *Task) GetDate(name string) time.Time {
	if value, ok := t.data[name]; ok {
		epoch, err := strconv.Atoi(value)
		if err != nil {
			return time.Time{}
		}
		return time.Unix(int64(epoch), 0).UTC()
	}
	return time.Time{}
}

// SetDate sets the given task attribute.
func (t *Task) SetDate(name string, d time.Time) {
	t.data[name] = fmt.Sprintf("%v", d.Unix())
}

// Has returns  true only if the task has the given attribute, it doesn't
// matter if it's set with the zero value.
func (t *Task) Has(name string) bool {
	_, ok := t.data[name]
	return ok
}

// GetAttrNames returns the list of task attribute names.
func (t *Task) GetAttrNames() []string {
	attrs := make([]string, 0, len(t.data))
	for k := range t.data {
		attrs = append(attrs, k)
	}
	return attrs
}

// Remove removes an attribute or does not do anything in case it doesn't
// exist.
func (t *Task) Remove(name string) {
	delete(t.data, name)
}

// ComposeJSON converts a given task to its JSON representation.  Decorate
// parameter allows including the "id" task attribute.
func (t *Task) ComposeJSON() string {
	filtered := make(map[string]interface{})

	for attrName, attrValue := range t.data {
		attrType := attributeTypes[attrName]

		if strings.HasPrefix(attrName, "annotation_") {
			epoch, err := strconv.Atoi(attrName[len("annotation_"):])
			if err != nil {
				log.Warnf("Malformed annotation %q: %v", attrName, err)
				continue
			}

			newAnnotation := map[string]string{
				"entry":       time.Unix(int64(epoch), 0).UTC().Format(DateLayout),
				"description": attrValue,
			}

			annotations, ok := filtered["annotations"]
			if !ok {
				filtered["annotations"] = []map[string]string{newAnnotation}
			} else {
				filtered["annotations"] = append(annotations.([]map[string]string), newAnnotation)
			}
		} else if attrType == "date" {
			filtered[attrName] = t.GetDate(attrName).Format(DateLayout)
		} else if attrType == "numeric" {
			filtered[attrName] = t.GetInt(attrName)
		} else if attrName == "tags" {
			filtered[attrName] = strings.Split(attrValue, ",")
		} else if attrName == "depends" {
			// taskwarrior has two possible type for it, string or array.
			// see https://github.com/GothenburgBitFactory/taskserver/blob/1aaa22452c2c656c5cdb8e017368e0848e54555d/src/Task.cpp#L935-L948
			// Set string and not list to be compliant with taskd 1.2.0 and tw 2.5.x
			// TODO be aware of the config property "json.depends.array"
			filtered[attrName] = strings.Split(attrValue, ",")
			filtered[attrName] = fmt.Sprintf("%v", attrValue)
		} else if len(attrValue) > 0 {
			filtered[attrName] = attrValue
		}
	}

	value, err := json.Marshal(filtered)
	if err != nil {
		// TODO return an error or just log it?
		log.Errorf("Error marshaling task: %v", err)
		return ""
	}
	return string(value)
}

func (t *Task) addTag(tag string) {
	var tags []string
	if len(t.data["tags"]) > 0 {
		tags = strings.Split(t.data["tags"], ",")
	}
	for _, t := range tags {
		if t == tag {
			// tag already exists, don't add it
			return
		}
	}
	tags = append(tags, tag)
	t.data["tags"] = strings.Join(tags, ",")
}

func (t *Task) addDependency(dependency string) error {
	if dependency == t.data["uuid"] {
		return fmt.Errorf("a task cannot be dependent on itself")
	}

	depends := t.data["depends"]
	if depends != "" {
		// Check for extant dependency.
		if !strings.Contains(depends, dependency) {
			t.data["depends"] = fmt.Sprintf("%s,%s", depends, dependency)
		}
	} else {
		t.data["depends"] = dependency
	}
	return nil
}

// Copy returns a copy of the task
func (t *Task) Copy() Task {
	ret := Task{
		annotationCount: t.annotationCount,
		data:            make(map[string]string),
	}

	for k, v := range t.data {
		ret.data[k] = v
	}

	return ret
}
