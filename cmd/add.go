package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/szaffarano/gotas/pkg/task/repo"
)

func addCmd() *cobra.Command {
	var addCmd = cobra.Command{
		Use:   "add",
		Short: "Creates a new organization or user.",
		Long: `When creating a new user, shows the resultant UUID that the client software
useѕ to uniquely identify a user, because <user-name> need not be unique.`,
	}

	var addOrgCmd = cobra.Command{
		Aliases: []string{"o"},
		Use:     "org <organization>",
		Short:   "Creates a new organization",
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

			org, err := repository.NewOrg(orgName)
			if err != nil {
				return err
			}

			log.Infof("created organization %q", org.Name)

			return nil
		},
	}

	var addUserCmd = cobra.Command{
		Aliases: []string{"u"},
		Use:     "user <organization> <user>",
		Short:   "Creates a new user",
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

			user, err := repository.AddUser(orgName, userName)
			if err != nil {
				return err
			}

			log.Infof("New user key: %v", user.Key)
			log.Infof("Created user %q for organization %q", user.Name, user.Org.Name)

			return nil
		},
	}

	addCmd.AddCommand(&addOrgCmd)
	addCmd.AddCommand(&addUserCmd)

	return &addCmd
}
