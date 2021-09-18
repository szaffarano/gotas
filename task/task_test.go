package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageParsing(t *testing.T) {
	cases := []struct {
		title    string
		given    string
		expected Message
		failure  bool
	}{
		{
			title:    "simple message with payload should work",
			given:    "type: sync\n\npayload",
			expected: Message{Header: map[string]string{"type": "sync"}, Payload: "payload"},
			failure:  false,
		},

		{
			title: "bigger message with payload should work",
			given: `type: response
client: taskd 1.0.0
protocol: v1
code: 200
status: Ok

45da7110-1bcc-4318-d33e-12267a774e0f`,
			expected: Message{
				Header: map[string]string{
					"type":     "response",
					"client":   "taskd 1.0.0",
					"protocol": "v1",
					"code":     "200",
					"status":   "Ok",
				},
				Payload: "45da7110-1bcc-4318-d33e-12267a774e0f",
			},
		},

		{
			title:   "malformed message should fail",
			given:   "type: response\n",
			failure: true,
		},

		{
			title:   "message with invalid separators should fail",
			given:   "type response\n\n",
			failure: true,
		},

		{
			title:    "message with empty payload should be parsed",
			given:    "type: response\n\n",
			expected: Message{Header: map[string]string{"type": "response"}},
			failure:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			message, err := NewMessage(c.given)
			if err != nil {
				assert.True(t, c.failure, err.Error())
			} else if err == nil {
				assert.False(t, c.failure, "Failure expected, got a message")
				if !c.failure {
					assert.Equal(t, c.expected, message)
				}
			}
		})
	}
}

func TestToStringMessage(t *testing.T) {
	cases := []struct {
		title    string
		given    Message
		expected string
	}{
		{
			title:    "simple message",
			given:    Message{Header: map[string]string{"type": "response"}},
			expected: "type: response\n\n",
		},
		{
			title:    "simple message with payload",
			given:    Message{map[string]string{"type": "response"}, "payload"},
			expected: "type: response\n\npayload",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, c.given.String())
	}
}
