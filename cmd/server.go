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

package cmd

import (
	"encoding/binary"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/config"
	"github.com/szaffarano/gotas/pkg/task"
)

func serverCmd() *cobra.Command {
	daemon := false
	var serverCmd = cobra.Command{
		Use:   "server",
		Short: "Runs the server",
		Run: func(_ *cobra.Command, _ []string) {
			server, err := task.NewServer()
			if err != nil {
				log.Fatalf("Error initializing server: %s", err.Error())
			}
			defer func() {
				if err := server.Close(); err != nil {
					log.Errorf("Error closing server: %w", err)
				}
			}()

			for {
				client, err := server.NextClient()
				if err != nil {
					log.Errorf("Error receiving client: %s", err.Error())
				}

				go process(client)
			}
		},
	}

	serverCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Runs server as a daemon")

	return &serverCmd
}

func process(client task.Client) {
	log.Info("Processing new client")

	defer client.Close()

	cfg := config.Get()

	buffer := make([]byte, cfg.Request.Limit)

	// first 4 bytes are the message size
	if num, err := client.Read(buffer[:4]); err != nil {
		log.Errorf("Error reading size: %s", err.Error())
		return
	} else if num != 4 {
		log.Errorf("Error reading size, expected 4 bytes, but got %d", num)
		return
	}
	messageSize := int(binary.BigEndian.Uint32(buffer[:4]))

	if messageSize > len(buffer) {
		log.Errorf("Message limit exceeded: %d", messageSize)
		// @TODO reply a proper message to the client
		return
	}

	// messageSize includes the first 4 bytes
	messageSize, err := client.Read(buffer[:messageSize-4])
	if err != nil {
		log.Errorf("Error reading: %s", err.Error())
		return
	}

	msg, err := task.NewMessage(string(buffer[:messageSize]))
	if err != nil {
		log.Errorf("Error parsing message", err)
		return
	}
	log.Info("Message received")
	log.Debug(msg.String())
	response := task.Message{
		Header: map[string]string{
			"type":   "response",
			"code":   "201",
			"status": "Ok",
		},
	}

	responseMessage := response.Serialize()

	if size, err := client.Write([]byte(responseMessage[:4])); err != nil {
		log.Errorf("Error writing response to the client: %w", err)
		return
	} else if size != 4 {
		log.Errorf("Error writing response to the client")
		return
	}

	if size, err := client.Write([]byte(responseMessage[4:])); err != nil {
		log.Errorf("Error writing response to the client: %w", err)
		return
	} else if size != 4 {
		log.Errorf("Error writing response to the client")
		return
	}

	log.Info("Finishing")
}
