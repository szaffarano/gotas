package task

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			readFile(t, "task.json"),
			true,
			map[string]string{
				"customField":           "value for custom field",
				"entry":                 "1633003050",
				"modified":              "1633179167",
				"uuid":                  "b04d7885-31ff-4992-b4fe-5cde1b41ca54",
				"status":                "pending",
				"tags":                  "tag1,tag2",
				"depends":               "b8a25aa7-fea9-4abf-a487-02eacd85bd58",
				"description":           "New task",
				"annotation_1633003241": "A small annotation",
				"annotation_1633003244": "A small annotation 2",
			},
		},
		{
			"invalid json fails",
			readFile(t, "invalid-task.json"),
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
			`a2b5f6fc-7285-75cc-90b9-abf624a8457e - [] [entry:1632687645 priority: project:] [1632722433:"A small annotation"] Some task`,
			false,
			nil,
		},
		{
			"format FF2 fails",
			`37beef88-c3f8-a1e9-1f49-0a4856f7af7d - [] [entry:1632721666 priority: project:] annotate A small annotation`,
			false,
			nil,
		},
		{
			"format FF1 fails",
			`X [someTag] [att:value] description`,
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

func TestDetermineVersion(t *testing.T) {
	cases := []struct {
		raw     string
		version int
	}{
		{
			`X [someTag] [att:value] description`,
			1,
		},
		{
			`37beef88-c3f8-a1e9-1f49-0a4856f7af7d - [] [entry:1632721666 priority: project:] annotate A small annotation`,
			2,
		},
		{
			`a2b5f6fc-7285-75cc-90b9-abf624a8457e - [] [entry:1632687645 priority: project:] [1632722433:"A small annotation"] Some task`,
			3,
		},
		{
			`[description:"Some task" entry:"1632659723" status:"pending" uuid:"6b5af5e0-466a-4355-99db-719b19a5dcd3"]`,
			4,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("version %d", c.version), func(t *testing.T) {
			actual := determineVersion(c.raw)

			assert.Equal(t, c.version, actual)
		})
	}
}

func readFile(t *testing.T, path string) string {
	content, err := ioutil.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		assert.FailNowf(t, "error reading %v: %v", path, err.Error())
	}
	return string(content)
}
