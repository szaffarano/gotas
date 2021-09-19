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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/apex/log"
)

const (
	SEP = "\n\n"
)

type Message struct {
	Header  map[string]string
	Payload string
}

func NewMessage(raw string) (Message, error) {
	message := Message{
		Header: map[string]string{},
	}

	parts := strings.Split(raw, SEP)
	if len(parts) == 2 {
		message.Payload = parts[1]
	} else if len(parts) == 1 {
		return message, errors.New("Message separator not found")
	}

	headers := strings.Split(parts[0], "\n")
	for _, header := range headers {
		splitted := strings.Split(header, ": ")
		if len(splitted) != 2 {
			return message, fmt.Errorf("error parsing header entry: %q", header)
		}

		message.Header[splitted[0]] = splitted[1]
	}

	return message, nil
}

func (m Message) String() string {
	var builder strings.Builder
	for h := range m.Header {
		fmt.Fprintf(&builder, "%s: %s\n", h, m.Header[h])
	}
	fmt.Fprintf(&builder, "\n%s", m.Payload)

	return builder.String()
}

func (m Message) Serialize() []byte {
	msg := m.String()
	size := uint32(len(msg) + 4)

	buffer := new(bytes.Buffer)
	if err := binary.Write(buffer, binary.BigEndian, size); err != nil {
		log.Error("Error writing message to the client")
	}

	if err := binary.Write(buffer, binary.BigEndian, []byte(msg)); err != nil {
		log.Error("Error writing message to the client")
	}
	return buffer.Bytes()
}
