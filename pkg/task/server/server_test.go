package server

import (
	"io"
	"testing"
)

type mockedClient struct {
	send    io.Reader
	receive io.Writer
	closed  bool
}

func (m *mockedClient) Read(buf []byte) (int, error) {
	return m.send.Read(buf)
}

func (m *mockedClient) Write(buf []byte) (int, error) {
	return m.receive.Write(buf)
}

func (m *mockedClient) Close() error {
	m.closed = true
	return nil
}

func TestProcessMessage(t *testing.T) {
}
