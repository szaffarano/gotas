package task

import (
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
			return message, fmt.Errorf("Error parsing header entry: %q", header)
		}

		message.Header[splitted[0]] = splitted[1]
	}

	return message, nil
}

func (m Message) String() string {
	var b strings.Builder
	for h := range m.Header {
		fmt.Fprintf(&b, "%s: %s\n", h, m.Header[h])
	}
	fmt.Fprintf(&b, "\n%s", m.Payload)

	return b.String()
}
