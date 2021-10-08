package server

import (
	"bufio"
	"encoding/binary"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/szaffarano/gotas/pkg/task/message"
	"github.com/szaffarano/gotas/pkg/task/task"
)

type mockClient struct {
	reader *strings.Reader
	writer *strings.Builder
	closed bool
}

type mockAuth struct {
}

type mockReadAppender struct {
	reader *strings.Reader
	writer *strings.Builder
}

func (c *mockClient) Read(buf []byte) (int, error) {
	return c.reader.Read(buf)
}

func (c *mockClient) Write(buf []byte) (int, error) {
	return c.writer.Write(buf)
}

func (c *mockClient) Close() error {
	c.closed = true
	return nil
}

func (a *mockAuth) Authenticate(orgName, userName, key string) (task.User, error) {
	return task.User{}, nil
}

func (ra *mockReadAppender) Read(user task.User) ([]string, error) {
	scanner := bufio.NewScanner(ra.reader)
	var result []string
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result, nil
}

func (ra *mockReadAppender) Append(user task.User, data []string) error {
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
		{"initial sync", "sent-msg-init", "tx-init-before.data", "replied-msg-init", "tx-init-after.data"},
		{"sync with empty task data", "sent-msg-empty-init", "tx-empty-init-before.data", "replied-msg-empty-init", "tx-empty-init-after.data"},
		{"modified custom field", "sent-msg-custom-field", "tx-modify-custom-field-before.data", "replied-msg-custom-field", "tx-modify-custom-field-after.data"},
		{"tag modified in two branches", "msg-sent-case01", "tx-case01-before.data", "msg-replied-case01", "tx-case01-after.data"},
		{"sync after tag modified in two branches", "msg-sent-case02", "tx-case02-before.data", "msg-replied-case02", "tx-case02-after.data"},
		{"annotate task", "msg-sent-case03", "tx-case03-before.data", "msg-replied-case03", "tx-case03-after.data"},
		{"task merged", "sent-msg-merged-task", "tx-merged-task-before.data", "replied-msg-merged-task", "tx-merged-task-after.data"},
		{"merge tag and custom field", "msg-sent-case04", "tx-case04-before.data", "msg-replied-case04", "tx-case04-after.data"},
		{"sync after merge tag and custom field", "msg-sent-case05", "tx-case05-before.data", "msg-replied-case05", "tx-case05-after.data"},
		{"modify tags concurrently", "msg-sent-case06", "tx-case06-before.data", "msg-replied-case06", "tx-case06-after.data"},
		{"merge modify tags concurrently", "msg-sent-case07", "tx-case07-before.data", "msg-replied-case07", "tx-case07-after.data"},
		{"modify tag and due concurrently", "msg-sent-case08", "tx-case08-before.data", "msg-replied-case08", "tx-case08-after.data"},
		{"merge modify tag and due concurrently", "msg-sent-case09", "tx-case09-before.data", "msg-replied-case09", "tx-case09-after.data"},
		{"no changes", "msg-sent-case11", "tx-case11-before.data", "msg-replied-case11", "tx-case11-after.data"},
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

	data, err := ioutil.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		t.Errorf(err.Error())
	}
	return data
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

func parseMsg(t *testing.T, raw string) message.Message {
	t.Helper()

	raw = string([]byte(raw)[4:])

	msg, err := message.NewMessage(raw)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	return msg
}

func collectTxs(t *testing.T, txs string) ([]task.Task, []string) {
	var tasks []task.Task
	var ids []string

	scanner := bufio.NewScanner(strings.NewReader(txs))
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "{") {
			task, err := task.NewTask(l)
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
