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
	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	var removeCmd = cobra.Command{
		Use:   "remove",
		Short: "Deletes an organization or user.  Permanently.",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}
	return &removeCmd
}
