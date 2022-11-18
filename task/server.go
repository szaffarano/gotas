package task

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/szaffarano/gotas/task/auth"
)

const (
	// RequestLimitInBytes is the maximum size allowed for an incoming message
	// TODO read this value from the configuration
	RequestLimitInBytes = 1048576
)

// Reader reads user transactions
type Reader interface {
	Read(user auth.User) ([]string, error)
}

// Appender appends new transactions for a given user
type Appender interface {
	Append(user auth.User, data []string) error
}

// ReadAppender groups the basic Read and Append taskd functionality.
type ReadAppender interface {
	Reader
	Appender
}

// Process processes a taskd client request
func Process(client io.ReadWriteCloser, auth auth.Authenticator, ra ReadAppender) {
	defer client.Close()

	var msg, resp Message
	var err error

	if msg, err = receiveMessage(client); err != nil {
		log.Errorf("Error parsing message: %v", err)
		// TODO receive error code in the error
		if err = replyMessage(client, NewResponseMessage("500", err.Error())); err != nil {
			log.Errorf("Error replying error message to the client: %v", err)
		}
		return
	}

	loggedUser, err := isValid(msg, auth)
	if err != nil {
		if err = replyMessage(client, NewResponseMessage("400", err.Error())); err != nil {
			log.Errorf("Error replying error message to the client: %v", err)
		}
		return
	}

	resp = processMessage(msg, loggedUser, ra)

	if err := replyMessage(client, resp); err != nil {
		log.Errorf("Error sending response message: %v", err)
		return
	}
}

func receiveMessage(client io.Reader) (msg Message, err error) {
	buffer := make([]byte, 4)

	if num, err := client.Read(buffer); err != nil || num != 4 {
		return msg, fmt.Errorf("reading size, read %v bytes, got %v", num, err)
	}

	messageSize := int(binary.BigEndian.Uint32(buffer[:4]))
	if messageSize > RequestLimitInBytes {
		return Message{}, errors.New("message size limit exceeded")
	}

	buffer = make([]byte, messageSize-4)

	if _, err := client.Read(buffer); err != nil {
		return msg, fmt.Errorf("reading client, got %v", err)
	}

	return NewMessage(string(buffer))
}

func processMessage(msg Message, user auth.User, ra ReadAppender) (resp Message) {
	switch t := msg.Header["type"]; t {
	case "sync":
		return sync(msg, user, ra)
	default:
		return NewResponseMessage("500", fmt.Sprintf("unknown message type: %q", t))
	}
}

func replyMessage(client io.Writer, resp Message) error {
	responseMessage := resp.Serialize()

	if size, err := client.Write([]byte(responseMessage[:4])); err != nil || size < 4 {
		return fmt.Errorf("writing size to the client, sent %v: %v", size, err)
	}

	if size, err := client.Write([]byte(responseMessage[4:])); err != nil || size < len(responseMessage)-4 {
		return fmt.Errorf("writing response to the client, sent %v: %v", size, err)
	}

	return nil
}

func isValid(msg Message, a auth.Authenticator) (auth.User, error) {
	userName := msg.Header["user"]
	key := msg.Header["key"]
	orgName := msg.Header["org"]

	// verify user credentials
	loggedUser, err := a.Authenticate(orgName, userName, key)
	if err != nil {
		return loggedUser, err
	}

	// verify protocol version
	if msg.Header["protocol"] != "v1" {
		return auth.User{}, fmt.Errorf("protocol not supported (%s)", msg.Header["protocol"])
	}

	// TODO verify redirect

	return loggedUser, nil
}

func sync(msg Message, user auth.User, ra ReadAppender) Message {
	var err error
	tx, clientData := getClientData(msg.Payload)
	serverData, err := ra.Read(user)
	if err != nil {
		log.Errorf("Error reading user dada: %v", err)
		return NewResponseMessage("500", "Error reading user data")
	}
	log.Infof("Loaded %v records", len(serverData))

	branchPoint := findBranchPoint(serverData, tx)
	if branchPoint == -1 {
		return NewResponseMessage("500", "Could not find the last sync transaction. Did you skip the 'task sync init' requirement?")
	}

	serverSubset, err := extractSubset(serverData, branchPoint)
	if err != nil {
		return NewResponseMessage("500", err.Error())
	}

	var newServerData, newClientData []string

	// Maintain a list of already-merged task UUIDs.
	alreadySeen := make(map[string]bool)
	var storeCount, mergeCount int

	// For each incoming task...
	for _, clientTask := range clientData {
		// TODO Validate task?
		uuid := clientTask.Get("uuid")

		// If task is in subset
		if taskContains(serverSubset, "uuid", uuid) {
			// Merging a task causes a complete scan, and that picks up all mods to
			// that same task.  Therefore, there is no need to re-process a UUID.
			if _, ok := alreadySeen[uuid]; ok {
				continue
			}

			alreadySeen[uuid] = true

			// Find common ancestor, prior to branch point
			commonAncestor, err := findCommonAncestor(serverData, branchPoint, uuid)
			if err != nil {
				return NewResponseMessage("500", err.Error())
			}

			// List the client-side modifications.
			clientMods := getClientMods(clientData, uuid)

			// List the server-side modifications.
			serverMods, err := getServerMods(serverData, uuid, commonAncestor)
			if err != nil {
				return NewResponseMessage("500", err.Error())
			}

			// Merge sort between clientMods and serverMods, patching ancestor.
			combined, err := NewTask(serverData[commonAncestor])
			if err != nil {
				return NewResponseMessage("500", err.Error())
			}

			mergeSort(clientMods, serverMods, combined)

			combinedJSON := combined.ComposeJSON()

			// Append combined task to client and server data, if not already there.
			newServerData = append(newServerData, (combinedJSON + "\n"))
			newClientData = append(newClientData, combinedJSON)
			mergeCount++
		} else {
			// Task not in subset, therefore can be stored unmodified.  Does not get
			// returned to client.
			newServerData = append(newServerData, (clientTask.ComposeJSON() + "\n"))
			storeCount++
		}
	}

	log.Infof("Stored %v tasks, merged %v tasks", storeCount, mergeCount)

	// New server data means a new sync key must be generated.  No new server data
	// means the most recent sync key is reused.
	newSyncKey := ""
	if len(newServerData) > 0 {
		newSyncKey = uuid.New().String()
		newServerData = append(newServerData, (newSyncKey + "\n"))
		log.Infof("New sync key %q", newSyncKey)

		// Append new_server_data to file.
		// append_server_data(org, password, newServerData)
		if err := ra.Append(user, newServerData); err != nil {
			return NewResponseMessage("500", err.Error())
		}
	} else {
		for i := len(serverData) - 1; i >= 0; i-- {
			if !strings.HasPrefix(serverData[i], "{") {
				newSyncKey = serverData[i]
				break
			}
		}
		log.Infof("Sync key %q still valid", newSyncKey)
	}

	out := Message{
		Payload: getResponsePayload(serverSubset, newClientData, newSyncKey),
		Header:  make(map[string]string),
	}

	// If there are changes, respond with 200, otherwise 201.
	if len(serverSubset) > 0 || len(newClientData) > 0 || len(newServerData) > 0 {
		log.Infof("returning 200")
		out.Header["code"] = "200"
		out.Header["status"] = ErrorCodes[200]
	} else {
		log.Infof("returning 201")
		out.Header["code"] = "201"
		out.Header["status"] = ErrorCodes[201]
		log.Infof("No change")
	}

	return out
}

func getResponsePayload(serverSubset []Task, newClientData []string, newSyncKey string) string {
	// If there is outgoing data, generate payload + key.
	payload := ""
	if len(serverSubset) > 0 || len(newClientData) > 0 {
		payload = generatePayload(serverSubset, newClientData, newSyncKey)
	} else {
		// No outgoing data, just sent the latest key.
		payload = newSyncKey + "\n"
	}

	return payload
}

func getClientData(payload string) (tx string, tasks []Task) {
	scanner := bufio.NewScanner(strings.NewReader(payload))
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 0 {
			if strings.HasPrefix(line, "{") {
				t, err := NewTask(line)
				if err != nil {
					log.Warnf("Error parsing task: %v", err)
					continue
				}
				tasks = append(tasks, t)

			} else {
				if parsed, err := uuid.Parse(line); err != nil {
					log.Warnf("Error parsing UUID %s: %v", line, err)
				} else {
					tx = parsed.String()
				}
			}
		}
	}
	return tx, tasks
}

func findBranchPoint(data []string, key string) int {
	// A missing key is either a first-time sync, or a request to get all data.
	if key == "" {
		return 0
	}

	for idx, value := range data {
		if value == key {
			log.Infof("Branch point: %s --> %d", key, idx)
			return idx
		}
	}
	log.Infof("Branch point not found: %s", key)

	return -1
}

func extractSubset(data []string, branchPoint int) ([]Task, error) {

	var tasks []Task
	if branchPoint < len(data) {
		tasks = make([]Task, 0, len(data)-branchPoint)
		for i := branchPoint; i < len(data); i++ {
			if strings.HasPrefix(data[i], "{") {
				t, err := NewTask(data[i])
				if err != nil {
					return nil, err
				}
				tasks = append(tasks, t)
			}
		}

	}
	log.Infof("Subset %v tasks", len(tasks))
	return tasks, nil
}

func taskContains(taskList []Task, name, value string) bool {
	for _, t := range taskList {
		if t.Get(name) == value {
			return true
		}
	}
	return false
}

func sliceContains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func findCommonAncestor(data []string, branchPoint int, uuid string) (int, error) {
	log.Infof("Finding commong ancestor for uuid = %s and branch point = %d", uuid, branchPoint)

	for i := branchPoint; i >= 0; i-- {
		log.Infof("Reading line to compare ancestor for uuid = %s and branch point = %s", uuid, data[i])

		if strings.HasPrefix(data[i], "{") {
			t, err := NewTask(data[i])
			if err != nil {
				return 0, err
			}
			log.Infof("Comparing common ancestor %s == %s", uuid, t.Get("uuid"))

			if t.Get("uuid") == uuid {
				log.Infof("Common ancestor found uuid = %s, idx = %d", uuid, i)

				return i, nil
			}
		}
	}

	return 0, fmt.Errorf("could not find common ancestor for %q. Did you skip the 'task sync init' requirement?", uuid)
}

// Extract tasks from the client list, with the given UUID, maintaining the
// sequence.
func getClientMods(data []Task, uuid string) []Task {
	var mods []Task
	for _, t := range data {
		if t.Get("uuid") == uuid {
			mods = append(mods, t)
		}
	}
	return mods
}

// Extract tasks from the server list, with the given UUID, maintaining the
// sequence.
func getServerMods(data []string, uuid string, ancestor int) ([]Task, error) {
	var mods []Task
	for i := ancestor + 1; i < len(data); i++ {
		if strings.HasPrefix(data[i], "{") {
			t, err := NewTask(data[i])
			if err != nil {
				return nil, err
			}
			if t.Get("uuid") == uuid {
				mods = append(mods, t)
			}
		}
	}
	return mods, nil
}

// Simultaneously walks two lists, select either the left or the right depending
// on last modification time.
func mergeSort(left []Task, right []Task, combined Task) {
	prevLeft, prevRight := combined.Copy(), combined.Copy()
	var idxLeft, idxRight int

	for idxLeft < len(left) && idxRight < len(right) {
		modLeft := lastModification(left[idxLeft])
		modRigth := lastModification(right[idxRight])
		if modLeft.Before(modRigth) {
			log.Infof("applying left %d < %d", modLeft.Unix(), modRigth.Unix())
			patch(combined, prevLeft, left[idxLeft])
			combined.SetDate("modified", modLeft)
			prevLeft = left[idxLeft]
			idxLeft++
		} else {
			log.Infof("applying right %d >= %d", modLeft.Unix(), modRigth.Unix())
			patch(combined, prevRight, right[idxRight])
			combined.SetDate("modified", modRigth)
			prevRight = right[idxRight]
			idxRight++
		}
	}

	for idxLeft < len(left) {
		patch(combined, prevLeft, left[idxLeft])
		combined.SetDate("modified", lastModification(left[idxLeft]))
		prevLeft = left[idxLeft]
		idxLeft++
	}

	for idxRight < len(right) {
		patch(combined, prevRight, right[idxRight])
		combined.SetDate("modified", lastModification(right[idxRight]))
		prevRight = right[idxRight]
		idxRight++
	}

	log.Infof("Merge result %s", combined.ComposeJSON())
}

// //////////////////////////////////////////////////////////////////////////////
// Get the last modication time for a task.  Ideally this is the attribute
// "modification".  If that is missing (pre taskwarrior 2.2.0), use the later of
// the "entry", "end", or"start" dates.
func lastModification(t Task) time.Time {
	dateFields := []string{"modified", "end", "start"}

	for _, f := range dateFields {
		if t.Has(f) {
			return t.GetDate(f)
		}
	}

	return t.GetDate("entry")
}

func generatePayload(subset []Task, additions []string, key string) string {
	payload := new(strings.Builder)

	for _, s := range subset {
		payload.Write([]byte(s.ComposeJSON()))
		payload.Write([]byte("\n"))
	}

	for _, a := range additions {
		payload.Write([]byte(a))
		payload.Write([]byte("\n"))
	}

	payload.Write([]byte(key))
	payload.Write([]byte("\n"))

	return payload.String()
}

// //////////////////////////////////////////////////////////////////////////////
// Determine the delta between 'from' and 'to', and apply only those changes to
// 'base'.  All three tasks have the same uuid.
func patch(base, from, to Task) {
	// Determine the different attribute names between from and to.
	fromAtts := from.GetAttrNames()
	toAtts := to.GetAttrNames()

	fromOnly, toOnly := listDiff(fromAtts, toAtts)
	commonAtts := listIntersect(fromAtts, toAtts)

	// The from-only attributes must be deleted from base.
	for _, att := range fromOnly {
		log.Infof("patch remove %v", att)
		base.Remove(att)
	}

	// The to-only attributes must be added to base.
	for _, att := range toOnly {
		log.Infof("patch add %v=%v", att, to.Get(att))
		base.Set(att, to.Get(att))
	}

	// The intersecting attributes, if the values differ, are applied.
	for _, att := range commonAtts {
		if from.Get(att) != to.Get(att) {
			log.Infof("patch modify %v=%v", att, to.Get(att))
			base.Set(att, to.Get(att))
		}
	}
}

// List operations.
func listDiff(left, right []string) (leftOnly, rightOnly []string) {

	for _, l := range left {
		if !sliceContains(right, l) {
			leftOnly = append(leftOnly, l)
		}
	}

	for _, r := range right {
		if !sliceContains(left, r) {
			rightOnly = append(rightOnly, r)
		}
	}

	return leftOnly, rightOnly
}

func listIntersect(left, right []string) (intersection []string) {
	for _, l := range left {
		if sliceContains(right, l) {
			intersection = append(intersection, l)
		}
	}

	return intersection
}
