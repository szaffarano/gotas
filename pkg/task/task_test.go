package task

import (
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
