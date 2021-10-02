package cmd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task/message"
	"github.com/szaffarano/gotas/pkg/task/repo"
	"github.com/szaffarano/gotas/pkg/task/task"
	server "github.com/szaffarano/gotas/pkg/task/transport"
)

func serverCmd() *cobra.Command {
	daemon := false
	var serverCmd = cobra.Command{
		Use:   "server",
		Short: "Runs the server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataDir := cmd.Flag(dataFlag).Value.String()

			configFilePath := filepath.Join(dataDir, "config")
			cfg, err := config.Load(configFilePath)
			if err != nil {
				return err
			}

			server, err := server.NewServer(cfg)
			if err != nil {
				return fmt.Errorf("initializing server: %v", err)
			}
			defer func() {
				if err := server.Close(); err != nil {
					panic(fmt.Sprintf("error closing server: %v", err))
				}
			}()

			// TODO graceful shutdown

			for {
				client, err := server.NextClient()
				if err != nil {
					log.Errorf("Error receiving client: %s", err.Error())
				}

				go process(client, cfg)
			}
		},
	}

	serverCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Runs server as a daemon")

	return &serverCmd
}

func process(client server.Client, cfg config.Config) {
	defer client.Close()

	msg, err := receiveMessage(client)
	if err != nil {
		log.Errorf("Error parsing message", err)
		return
	}

	log.Debugf("Message received: %v", msg)

	resp, err := processMessage(msg, cfg)
	if err != nil {
		log.Errorf("Error handling input message: %v", err)
		return
	}

	if err := replyMessage(client, resp); err != nil {
		log.Errorf("Error sending response message: %v", err)
		return
	}

	log.Debugf("Replying message: %v", resp)
}

func receiveMessage(client io.Reader) (msg message.Message, err error) {
	buffer := make([]byte, 4)

	if num, err := client.Read(buffer); err != nil {
		return msg, fmt.Errorf("reading size: %v", err)
	} else if num != 4 {
		return msg, fmt.Errorf("reading size, expected 4 bytes, but got %d", num)
	}

	messageSize := int(binary.BigEndian.Uint32(buffer[:4]))
	buffer = make([]byte, messageSize)

	if _, err := client.Read(buffer); err != nil {
		return msg, fmt.Errorf("reading client: %v", err)
	}

	// TODO verify request limit

	return message.NewMessage(string(buffer))
}

func processMessage(msg message.Message, cfg config.Config) (resp message.Message, err error) {
	switch t := msg.Header["type"]; t {
	case "sync":
		return sync(msg, cfg)
	default:
		resp = message.Message{
			Header: map[string]string{
				"type":   "response",
				"code":   "500",
				"status": fmt.Sprintf("unexpected message type: %q", t),
			},
		}
	}

	return resp, nil
}

func replyMessage(client io.Writer, resp message.Message) error {
	responseMessage := resp.Serialize()
	if size, err := client.Write([]byte(responseMessage[:4])); err != nil {
		return fmt.Errorf("writing response size to the client: %v", err)
	} else if size < 4 {
		return fmt.Errorf("incomplete response sent to the client, only %v bytes", size)
	}

	if size, err := client.Write([]byte(responseMessage[4:])); err != nil {
		return fmt.Errorf("writing response to the client: %v", err)
	} else if size < len(responseMessage)-4 {
		return fmt.Errorf("incomplete response sent to the client, only %v bytes", size)
	}
	return nil
}

func sync(msg message.Message, cfg config.Config) (message.Message, error) {
	// verify user credentials
	repository, err := repo.OpenRepository(cfg.Get(repo.Root))
	if err != nil {
		return message.Message{
			Header: map[string]string{
				"type":   "response",
				"code":   "500",
				"status": "Error opening repository",
			},
		}, err
	}

	org, err := repository.GetOrg(msg.Header["org"])
	if err != nil {
		// TODO more specific errors?
		return message.Message{
			Header: map[string]string{
				"type":   "response",
				"code":   "400",
				"status": "Invalid org",
			},
		}, err
	}

	found := false
	for _, u := range org.Users {
		if u.Key == msg.Header["key"] && u.Name == msg.Header["user"] {
			found = true
			break
		}
	}
	if !found {
		// TODO more specific errors?
		return message.Message{
			Header: map[string]string{
				"type":   "response",
				"code":   "401",
				"status": "Invalid username or key",
			},
		}, err
	}

	// verify v1
	if msg.Header["protocol"] != "v1" {
		return message.Message{
			Header: map[string]string{
				"type":   "response",
				"code":   "400",
				"status": "Protocol not supported",
			},
		}, err
	}

	// TODO verify redirect

	scanner := bufio.NewScanner(strings.NewReader(msg.Payload))

	var tx *uuid.UUID
	var tasks []task.Task
	for scanner.Scan() {
		line := scanner.Text()

		if tx == nil {
			if t, err := uuid.Parse(line); err == nil {
				tx = &t
				continue
			}
		}
		t, err := task.NewTask(line)
		if err != nil {
			log.Errorf("Error parsing task: %v", err)
			continue
		}
		tasks = append(tasks, t)
	}

	log.Infof("Msg sync, tx = %v, tasks = %v", tx, tasks)

	resp := message.Message{
		Header: map[string]string{
			"type":   "response",
			"code":   "201",
			"status": "Ok",
		},
	}

	return resp, nil
}
