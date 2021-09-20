// Copyright © 2021 Sebastián Zaffarano <sebas@zaffarano.com.ar>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"encoding/binary"
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
func TestSerializeMessage(t *testing.T) {
	cases := []struct {
		title    string
		given    Message
		expected []byte
	}{
		{
			title:    "simple message",
			given:    Message{Header: map[string]string{"type": "response"}},
			expected: []byte("type: response\n\n"),
		},
		{
			title:    "simple message with payload",
			given:    Message{map[string]string{"type": "response"}, "payload"},
			expected: []byte("type: response\n\npayload"),
		},
	}

	for _, c := range cases {
		message := c.given.Serialize()
		size := binary.BigEndian.Uint32(message[:4])
		assert.Equal(t, c.expected, message[4:])
		assert.Equal(t, uint32(len(message)), size)
	}
}
