package task

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/task/auth"
)

type mockClient struct {
	reader     *strings.Reader
	writer     *strings.Builder
	closed     bool
	failReader bool
	failWriter bool
}

type mockAuth struct {
	fails bool
}

type mockReadAppender struct {
	reader *strings.Reader
	writer *strings.Builder
}

func (c *mockClient) Read(buf []byte) (int, error) {
	if c.failReader {
		return 0, errors.New("Error reading")
	}
	return c.reader.Read(buf)
}

func (c *mockClient) Write(buf []byte) (int, error) {
	if c.failWriter {
		return 0, errors.New("Error reading")
	}
	return c.writer.Write(buf)
}

func (c *mockClient) Close() error {
	c.closed = true
	return nil
}

func (a *mockAuth) Authenticate(orgName, userName, key string) (auth.User, error) {
	if a.fails {
		return auth.User{}, errors.New("Invalid credentials")
	}
	return auth.User{}, nil
}

func (ra *mockReadAppender) Read(user auth.User) ([]string, error) {
	scanner := bufio.NewScanner(ra.reader)
	var result []string
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result, nil
}

func (ra *mockReadAppender) Append(user auth.User, data []string) error {
	for _, d := range data {
		ra.writer.Write([]byte(d))
	}
	return nil
}

func TestProcessMessage(t *testing.T) {

	cases := []struct {
		title      string
		msgSent    string
		txBefore   string
		msgReplied string
		txAfter    string
	}{
		{"initial sync", "msg-sent-init", "tx-init-before.data", "msg-replied-init", "tx-init-after.data"},
		{"sync with empty task data", "msg-sent-empty-init", "tx-empty-init-before.data", "msg-replied-empty-init", "tx-empty-init-after.data"},
		{"modified custom field", "msg-sent-custom-field", "tx-modify-custom-field-before.data", "msg-replied-custom-field", "tx-modify-custom-field-after.data"},
		{"tag modified in two branches", "msg-sent-case01", "tx-case01-before.data", "msg-replied-case01", "tx-case01-after.data"},
		{"sync after tag modified in two branches", "msg-sent-case02", "tx-case02-before.data", "msg-replied-case02", "tx-case02-after.data"},
		{"annotate task", "msg-sent-case03", "tx-case03-before.data", "msg-replied-case03", "tx-case03-after.data"},
		{"task merged", "msg-sent-merged-task", "tx-merged-task-before.data", "msg-replied-merged-task", "tx-merged-task-after.data"},
		{"merge tag and custom field", "msg-sent-case04", "tx-case04-before.data", "msg-replied-case04", "tx-case04-after.data"},
		{"sync after merge tag and custom field", "msg-sent-case05", "tx-case05-before.data", "msg-replied-case05", "tx-case05-after.data"},
		{"modify tags concurrently", "msg-sent-case06", "tx-case06-before.data", "msg-replied-case06", "tx-case06-after.data"},
		{"merge modify tags concurrently", "msg-sent-case07", "tx-case07-before.data", "msg-replied-case07", "tx-case07-after.data"},
		{"modify tag and due concurrently", "msg-sent-case08", "tx-case08-before.data", "msg-replied-case08", "tx-case08-after.data"},
		{"merge modify tag and due concurrently", "msg-sent-case09", "tx-case09-before.data", "msg-replied-case09", "tx-case09-after.data"},
		{"no changes", "msg-sent-case11", "tx-case11-before.data", "msg-replied-case11", "tx-case11-after.data"},
		{"invalid protocol", "msg-sent-invalid-protocol", "empty-tx", "msg-replied-invalid-protocol", "empty-tx"},
	}

	for _, c := range cases {

		t.Run(c.title, func(t *testing.T) {
			txBeforeContent := loadFile(t, c.txBefore)

			client := &mockClient{
				reader: strings.NewReader(loadPayload(t, c.msgSent)),
				writer: new(strings.Builder),
			}

			auth := &mockAuth{}
			ra := &mockReadAppender{
				reader: strings.NewReader(string(txBeforeContent)),
				writer: new(strings.Builder),
			}
			ra.writer.Write(txBeforeContent)

			expected := loadFile(t, c.txAfter)

			Process(client, auth, ra)

			assert.True(t, client.closed)
			assert.NotNil(t, client.writer.String())

			compareTx(t, string(expected), ra.writer.String())
			comparePayloads(t, loadPayload(t, c.msgReplied), client.writer.String())
		})
	}

	t.Run("fail if reader fails", func(t *testing.T) {
		client := &mockClient{
			writer:     new(strings.Builder),
			failReader: true,
		}
		auth := &mockAuth{}
		ra := &mockReadAppender{
			writer: new(strings.Builder),
		}

		Process(client, auth, ra)

		comparePayloads(t, string(loadPayload(t, "msg-replied-error-reading")), client.writer.String())
	})

	t.Run("fail if client broken pipe", func(t *testing.T) {
		client := &mockClient{
			writer: new(strings.Builder),
			reader: strings.NewReader(loadPayload(t, "tx-init-before.data")),
		}
		auth := &mockAuth{}
		ra := &mockReadAppender{
			writer: new(strings.Builder),
		}

		Process(client, auth, ra)

		comparePayloads(t, string(loadPayload(t, "msg-replied-client-broken-pipe")), client.writer.String())
	})

	t.Run("fail if invalid credentials", func(t *testing.T) {
		client := &mockClient{
			writer: new(strings.Builder),
			reader: strings.NewReader(loadPayload(t, "msg-sent-init")),
		}
		auth := &mockAuth{fails: true}
		ra := &mockReadAppender{
			writer: new(strings.Builder),
		}

		Process(client, auth, ra)

		comparePayloads(t, string(loadPayload(t, "msg-replied-invalid-credentials")), client.writer.String())
	})

	t.Run("fail if writer fails", func(t *testing.T) {
		client := &mockClient{
			writer:     new(strings.Builder),
			failReader: true,
			failWriter: true,
		}
		auth := &mockAuth{}
		ra := &mockReadAppender{
			writer: new(strings.Builder),
		}

		Process(client, auth, ra)

		assert.Equal(t, 0, len(client.writer.String()))
	})

	t.Run("fail if size exceeded", func(t *testing.T) {
		sizeBuffer := make([]byte, 4)
		binary.BigEndian.PutUint32(sizeBuffer, uint32(RequestLimitInBytes+1))

		client := &mockClient{
			reader: strings.NewReader(string(sizeBuffer)),
			writer: new(strings.Builder),
		}

		auth := &mockAuth{}
		ra := &mockReadAppender{
			writer: new(strings.Builder),
		}

		Process(client, auth, ra)

		comparePayloads(t, string(loadPayload(t, "msg-replied-size-exceeded")), client.writer.String())
	})
}

func loadPayload(t *testing.T, path string) string {
	t.Helper()

	data := loadFile(t, path)
	size := uint32(len(data) + 4)

	buffer := make([]byte, size)

	binary.BigEndian.PutUint32(buffer[:4], size)
	copy(buffer[4:], data)

	return string(buffer)
}

func loadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", "payloads", path))
	if err != nil {
		t.Errorf(err.Error())
	}
	return normalizeNewlines(data)
}

func normalizeNewlines(d []byte) []byte {
	// replace CR LF \r\n (windows) with LF \n (unix)
	d = bytes.ReplaceAll(d, []byte{13, 10}, []byte{10})
	// replace CF \r (mac) with LF \n (unix)
	d = bytes.ReplaceAll(d, []byte{13}, []byte{10})
	return d
}

func compareTx(t *testing.T, expected, actual string) {
	tasksExpected, idsExpected := collectTxs(t, expected)
	tasksActual, idsActual := collectTxs(t, actual)

	assert.Equal(t, tasksExpected, tasksActual)
	// tx ids are uuid, how to mock them? So far, just expect the same number of ids
	assert.Equal(t, len(idsExpected), len(idsActual))
}

func comparePayloads(t *testing.T, expected, actual string) {
	t.Helper()

	if assert.Greater(t, len(actual), 0) {

		expMsg := parseMsg(t, expected)
		actMsg := parseMsg(t, actual)

		assert.Equal(t, expMsg.Header, actMsg.Header)
	}
}

func parseMsg(t *testing.T, raw string) Message {
	t.Helper()

	raw = string([]byte(raw)[4:])

	msg, err := NewMessage(raw)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	return msg
}

func collectTxs(t *testing.T, txs string) ([]Task, []string) {
	var tasks []Task
	var ids []string

	scanner := bufio.NewScanner(strings.NewReader(txs))
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "{") {
			task, err := NewTask(l)
			if err != nil {
				assert.FailNow(t, err.Error())
			}
			tasks = append(tasks, task)
		} else {
			ids = append(ids, l)
		}
	}

	return tasks, ids
}
