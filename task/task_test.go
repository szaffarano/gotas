package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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
			"json format fails (not implemented)",
			readFile(t, "task-tags-as-string.json"),
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
			"json format fails (not implemented)",
			readFile(t, "task-2.json"),
			true,
			map[string]string{
				"customField":           "value for custom field",
				"entry":                 "1633003050",
				"modified":              "1633179167",
				"uuid":                  "b04d7885-31ff-4992-b4fe-5cde1b41ca54",
				"imask":                 "1",
				"status":                "pending",
				"tags":                  "tag1,tag2",
				"depends":               "abc,xyz",
				"description":           "New task",
				"annotation_1633003241": "A small annotation",
				"annotation_1633003244": "A small annotation 2",
			},
		},
		{"task depends itself", readFile(t, "task-invalid-depends-itself.json"), false, nil},
		{"task depends itself when depends is slice", readFile(t, "task-invalid-depends-itself-2.json"), false, nil},
		{"task invalid entry date", readFile(t, "task-invalid-entry-date.json"), false, nil},
		{"task invalid modification date", readFile(t, "task-invalid-modif-date.json"), false, nil},
		{"task malformed json", readFile(t, "invalid-task.json"), false, nil},
		{"task invalid tags", readFile(t, "task-invalid-tags.json"), false, nil},
		{"task invalid depends", readFile(t, "task-invalid-depends.json"), false, nil},
		{"task invalid annotation type", readFile(t, "task-invalid-annotation-type.json"), false, nil},
		{"task invalid annotation entry type", readFile(t, "task-invalid-annotation-entry-type.json"), false, nil},
		{"task invalid annotation entry date", readFile(t, "task-invalid-annotation-entry-date.json"), false, nil},
		{"task invalid annotation desc type", readFile(t, "task-invalid-annotation-desc-type.json"), false, nil},
		{"task invalid annotation desc format", readFile(t, "task-invalid-annotation-desc-date.json"), false, nil},
		{"empty string fails", "", false, nil},
		{"string with invalid rune fails", "\xbd\xb2", false, nil},
		{"format FF3 fails", `a2b5f6fc-7285-75cc-90b9-abf624a8457e - [] [entry:1632687645 priority: project:] [1632722433:"A small annotation"] Some task`, false, nil},
		{"format FF2 fails", `37beef88-c3f8-a1e9-1f49-0a4856f7af7d - [] [entry:1632721666 priority: project:] annotate A small annotation`, false, nil},
		{"format FF1 fails", `X [someTag] [att:value] description`, false, nil},
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

	t.Run("task compose json", func(t *testing.T) {
		task, err := NewTask(readFile(t, "task-2.json"))
		assert.Nil(t, err)

		json := task.ComposeJSON()
		task2, err := NewTask(json)
		assert.Nil(t, err)

		assert.Equal(t, task, task2)
	})

	t.Run("gets and sets", func(t *testing.T) {
		task, err := NewTask(readFile(t, "task-2.json"))
		assert.Nil(t, err)

		t.Run("attr names", func(t *testing.T) {
			assert.Greater(t, len(task.GetAttrNames()), 0)
		})

		t.Run("add new attribute", func(t *testing.T) {
			attrs := task.GetAttrNames()

			task.Set("newattr", "newvalue")
			attrsAfter := task.GetAttrNames()

			assert.Greater(t, len(attrsAfter), len(attrs))
			assert.Equal(t, task.Get("newattr"), "newvalue")
		})

		t.Run("get invalid integer attribute", func(t *testing.T) {
			assert.Equal(t, task.GetInt("newattr"), 0)
			assert.Equal(t, task.GetInt("invalid"), 0)
		})

		t.Run("invalid date attribute", func(t *testing.T) {
			assert.Equal(t, task.GetDate("newattr"), time.Time{})
			assert.Equal(t, task.GetDate("invalid"), time.Time{})
		})

		t.Run("has attribute", func(t *testing.T) {
			assert.True(t, task.Has("newattr"))
			assert.False(t, task.Has("invalid"))
		})

		t.Run("valid date attribute", func(t *testing.T) {
			now := time.Now().UTC()
			task.SetDate("newattr", now)
			assert.Equal(t, task.GetDate("newattr").Unix(), now.Unix())
		})

		t.Run("valid int attribute", func(t *testing.T) {
			task.Set("newattr", "99")
			assert.Equal(t, task.GetInt("newattr"), 99)
		})

		t.Run("remove attribute", func(t *testing.T) {
			attrsBefore := task.GetAttrNames()
			task.Remove("newattr")
			attrsAfter := task.GetAttrNames()
			assert.Equal(t, len(attrsAfter), len(attrsBefore)-1)
		})
	})

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
	content, err := os.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		assert.FailNowf(t, "error reading %v: %v", path, err.Error())
	}
	return string(content)
}
