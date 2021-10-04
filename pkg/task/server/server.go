package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/google/uuid"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/message"
	"github.com/szaffarano/gotas/pkg/task/repo"
	"github.com/szaffarano/gotas/pkg/task/task"
	"github.com/szaffarano/gotas/pkg/task/transport"
)

// Process processes a taskd client request
func Process(client transport.Client, cfg config.Config) {
	defer client.Close()

	var msg, resp message.Message
	var err error

	if msg, err = receiveMessage(client); err != nil {
		log.Errorf("Error parsing message", err)
		return
	}

	resp = processMessage(msg, cfg)

	if err := replyMessage(client, resp); err != nil {
		log.Errorf("Error sending response message: %v", err)
		return
	}
}

func receiveMessage(client io.Reader) (msg message.Message, err error) {
	buffer := make([]byte, 4)

	if num, err := client.Read(buffer); err != nil || num != 4 {
		return msg, fmt.Errorf("reading size, read %v bytes: %v", num, err)
	}

	messageSize := int(binary.BigEndian.Uint32(buffer[:4]))
	buffer = make([]byte, messageSize)

	if _, err := client.Read(buffer); err != nil {
		return msg, fmt.Errorf("reading client: %v", err)
	}

	// TODO verify request limit

	return message.NewMessage(string(buffer))
}

func processMessage(msg message.Message, cfg config.Config) (resp message.Message) {
	switch t := msg.Header["type"]; t {
	case "sync":
		return sync(msg, cfg)
	default:
		return message.NewResponseMessage("500", fmt.Sprintf("unknown message type: %q", t))
	}
}

func replyMessage(client io.Writer, resp message.Message) error {
	responseMessage := resp.Serialize()

	if size, err := client.Write([]byte(responseMessage[:4])); err != nil || size < 4 {
		return fmt.Errorf("writing size to the client, sent %v: %v", size, err)
	}

	if size, err := client.Write([]byte(responseMessage[4:])); err != nil || size < len(responseMessage)-4 {
		return fmt.Errorf("writing response to the client, sent %v: %v", size, err)
	}

	return nil
}

func sync(msg message.Message, cfg config.Config) message.Message {
	var loggedUser repo.User
	userName := msg.Header["user"]
	key := msg.Header["key"]
	orgName := msg.Header["org"]

	// verify user credentials
	repository, err := repo.OpenRepository(cfg.Get(repo.Root))
	if err != nil {
		log.Errorf("Error opening the repository: %v", err)
		return message.NewResponseMessage("500", "Error opening the repository")
	}

	if loggedUser, err = repository.Authenticate(orgName, userName, key); err != nil {
		code := "500"
		if authError, ok := err.(repo.AuthenticationError); ok {
			code = authError.Code
		}
		return message.NewResponseMessage(code, err.Error())
	}

	// verify protocol version
	if msg.Header["protocol"] != "v1" {
		return message.NewResponseMessage("400", "Protocol not supported")
	}

	// TODO verify redirect

	tx, tasks := clientData(msg.Payload)
	serverData, err := repository.GetData(loggedUser)
	if err != nil {
		log.Errorf("Error reading user dada: %v", err)
		return message.NewResponseMessage("500", "Error reading user data")
	}

	branchPoint := findBranchPoint(serverData, tx.String())
	if branchPoint == -1 {
		return message.NewResponseMessage("500", "Could not find the last sync transaction. Did you skip the 'task sync init' requirement?")
	}

	serverSubset, err := extractSubset(serverData, branchPoint)
	if err != nil {
		return message.NewResponseMessage("500", err.Error())
	}

	var newServerData, newClientData []string

	// Maintain a list of already-merged task UUIDs.
	alreadySeen := make(map[string]bool)
	var storeCount, mergeCount int

	// For each incoming task...
	for _, clientTask := range tasks {
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
			log.Infof("Common ancestor: %v", commonAncestor)
			if err != nil {
				return message.NewResponseMessage("500", err.Error())
			}

			// List the client-side modifications.
			clientMods := getClientMods(tasks, uuid)

			// List the server-side modifications.
			serverMods, err := getServerMods(serverData, uuid, commonAncestor)
			if err != nil {
				return message.NewResponseMessage("500", err.Error())
			}

			// Merge sort between clientMods and serverMods, patching ancestor.
			combined, err := task.NewTask(serverData[commonAncestor])
			if err != nil {
				return message.NewResponseMessage("500", err.Error())
			}

			mergeSort(clientMods, serverMods, combined)

			combinedJSON := combined.ComposeJSON(false)

			// Append combined task to client and server data, if not already there.
			newServerData = append(newServerData, (combinedJSON + "\n"))
			newClientData = append(newClientData, combinedJSON)
			mergeCount++
		} else {
			// Task not in subset, therefore can be stored unmodified.  Does not get
			// returned to client.
			newServerData = append(newServerData, (clientTask.ComposeJSON(false) + "\n"))
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
		if err := repository.AppendData(loggedUser, newServerData); err != nil {
			return message.NewResponseMessage("500", err.Error())
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

	out := message.Message{
		Payload: getResponsePayload(serverSubset, newClientData, newSyncKey),
		Header:  make(map[string]string),
	}

	// If there are changes, respond with 200, otherwise 201.
	if len(serverSubset) > 0 || len(newClientData) > 0 || len(newServerData) > 0 {
		log.Infof("returning 200")
		out.Header["code"] = "200"
		out.Header["status"] = task.ErrorCodes["200"]
	} else {
		log.Infof("returning 201")
		out.Header["code"] = "201"
		out.Header["status"] = task.ErrorCodes["201"]
		log.Infof("No change")
	}

	return out
}

func getResponsePayload(serverSubset []task.Task, newClientData []string, newSyncKey string) string {
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

func clientData(payload string) (tx uuid.UUID, tasks []task.Task) {
	var err error
	scanner := bufio.NewScanner(strings.NewReader(payload))
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) > 0 {
			if strings.HasPrefix(line, "{") {
				t, err := task.NewTask(line)
				if err != nil {
					log.Warnf("Error parsing task: %v", err)
					continue
				}
				tasks = append(tasks, t)

			} else {
				if tx, err = uuid.Parse(line); err != nil {
					log.Warnf("Error parsing UUID: %v", err)
				}
			}
		}
	}
	return tx, tasks
}

func findBranchPoint(data []string, key string) int {
	// A missing key is either a first-time sync, or a request to get all data.
	if key == "" || key == "00000000-0000-0000-0000-000000000000" {
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

func extractSubset(data []string, branchPoint int) ([]task.Task, error) {

	if branchPoint < len(data) {
		tasks := make([]task.Task, 0, len(data)-branchPoint)
		for i := branchPoint; i < len(data); i++ {
			if strings.HasPrefix(data[i], "{") {
				t, err := task.NewTask(data[i])
				if err != nil {
					return nil, err
				}
				tasks = append(tasks, t)
			}
		}

		return tasks, nil
	}
	return nil, fmt.Errorf("invalid branchPoint: %d for %d data length", branchPoint, len(data))
}

func taskContains(taskList []task.Task, name, value string) bool {
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
	for i := branchPoint; i >= 0; i++ {
		if strings.HasPrefix(data[i], "{") {
			t, err := task.NewTask(data[i])
			if err != nil {
				return 0, err
			}
			if t.Get("uuid") == uuid {
				return i, nil
			}
		}
	}

	return 0, fmt.Errorf("could not find common ancestor for %q. Did you skip the 'task sync init' requirement?", uuid)
}

// Extract tasks from the client list, with the given UUID, maintaining the
// sequence.
func getClientMods(data []task.Task, uuid string) []task.Task {
	var mods []task.Task
	for _, t := range data {
		if t.Get("uuid") == uuid {
			mods = append(mods, t)
		}
	}
	return mods
}

// Extract tasks from the server list, with the given UUID, maintaining the
// sequence.
func getServerMods(data []string, uuid string, ancestor int) ([]task.Task, error) {
	var mods []task.Task
	for i := ancestor + 1; i < len(data); i++ {
		if strings.HasPrefix(data[i], "{") {
			t, err := task.NewTask(data[i])
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
func mergeSort(left []task.Task, right []task.Task, combined task.Task) {
	dummy := []task.Task{combined}
	var prevLeft, idxLeft, prevRight, idxRight int

	for idxLeft < len(left) && idxRight < len(right) {
		modLeft := lastModification(dummy[idxLeft])
		modRigth := lastModification(right[idxRight])
		if modLeft.Before(modRigth) {
			log.Infof("applying left %v < %v", modLeft, modRigth)
			patch(combined, dummy[prevLeft], left[idxLeft])
			combined.SetDate("modified", modLeft)
			prevLeft = idxLeft
			idxLeft++
		} else {
			log.Infof("applying right %v >= %v", modLeft, modRigth)
			patch(combined, dummy[prevRight], right[idxRight])
			combined.SetDate("modified", modRigth)
			prevRight = idxRight
			idxRight++
		}
	}

	for idxLeft < len(left) {
		patch(combined, dummy[prevLeft], left[idxLeft])
		combined.SetDate("modified", lastModification(left[idxLeft]))
		prevLeft = idxLeft
		idxLeft++
	}

	for idxRight < len(right) {
		patch(combined, dummy[prevRight], right[idxRight])
		combined.SetDate("modified", lastModification(right[idxRight]))
		prevRight = idxRight
		idxRight++
	}

	log.Infof("Merge result {2}", combined.ComposeJSON(false))
}

////////////////////////////////////////////////////////////////////////////////
// Get the last modication time for a task.  Ideally this is the attribute
// "modification".  If that is missing (pre taskwarrior 2.2.0), use the later of
// the "entry", "end", or"start" dates.
func lastModification(t task.Task) time.Time {
	dateFields := []string{"modified", "end", "start"}

	for _, f := range dateFields {
		if t.Has(f) {
			return t.GetDate(f)
		}
	}

	return t.GetDate("entry")
}

func generatePayload(subset []task.Task, additions []string, key string) string {
	payload := new(strings.Builder)

	for _, s := range subset {
		payload.Write([]byte(s.ComposeJSON(false)))
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

////////////////////////////////////////////////////////////////////////////////
// Determine the delta between 'from' and 'to', and apply only those changes to
// 'base'.  All three tasks have the same uuid.
func patch(base, from, to task.Task) {
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
