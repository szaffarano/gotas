package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	SEP = "\n\n"
)

type Message struct {
	Header  map[string]string
	Payload string
}

// NewMessage parses a message
func NewMessage(raw string) (Message, error) {
	message := Message{
		Header: map[string]string{},
	}

	parts := strings.Split(raw, SEP)
	if len(parts) == 1 {
		return message, errors.New("Message separator not found")
	} else if len(parts) >= 2 {
		message.Payload = parts[1]
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

// NewResponseMessage is a helper method to create a simple response message
// with an initial header
func NewResponseMessage(code, status string) Message {
	return Message{
		Header: map[string]string{
			"type":   "response",
			"code":   code,
			"status": status,
		},
	}
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

	buffer := make([]byte, size)

	binary.BigEndian.PutUint32(buffer[:4], size)

	copy(buffer[4:], msg)

	return buffer
}
