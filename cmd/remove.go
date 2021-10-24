package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/task/repo"
)

func removeCmd() *cobra.Command {
	removeCmd := cobra.Command{
		Use:   "remove",
		Short: "Deletes an organization or user.  Permanently.",
		Run: func(_ *cobra.Command, _ []string) {
			log.Info("not implemented")
		},
	}

	removeOrgCmd := cobra.Command{
		Aliases: []string{"o"},
		Use:     "org <organization>",
		Short:   "Deletes a new organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				if err := cmd.Usage(); err != nil {
					return nil
				}
				return fmt.Errorf("organization name expected")
			}
			orgName := args[0]

			dataDir := cmd.Flag(dataFlag).Value.String()

			repository, err := repo.OpenRepository(dataDir)
			if err != nil {
				return err
			}

			err = repository.DelOrg(orgName)
			if err != nil {
				return err
			}

			log.Infof("removed organization %q", orgName)

			return nil
		},
	}

	removeUserCmd := cobra.Command{
		Aliases: []string{"u"},
		Use:     "user <organization> <user>",
		Short:   "Deletes a new user.  Users are identified by uuid, not name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				if err := cmd.Usage(); err != nil {
					return nil
				}
				return fmt.Errorf("organization and user name expected")
			}
			orgName := args[0]
			userName := args[1]

			dataDir := cmd.Flag(dataFlag).Value.String()
			repository, err := repo.OpenRepository(dataDir)
			if err != nil {
				return err
			}

			err = repository.DelUser(orgName, userName)
			if err != nil {
				return err
			}

			log.Infof("New user key: %v", userName)
			log.Infof("removed user %q from organization %q", userName, orgName)

			return nil
		},
	}

	removeCmd.AddCommand(&removeOrgCmd)
	removeCmd.AddCommand(&removeUserCmd)

	return &removeCmd
}
