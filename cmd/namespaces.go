package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wantedly/valec/aws"
)

// namespacesCmd represents the namespaces command
var namespacesCmd = &cobra.Command{
	Use:     "namespaces",
	Aliases: []string{"ns"},
	Short:   "List all namespaces",
	RunE:    doNamespaces,
}

func doNamespaces(cmd *cobra.Command, args []string) error {
	namespaces, err := aws.DynamoDB.ListNamespaces(rootOpts.tableName)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve namespaces.")
	}

	for _, namespace := range namespaces {
		fmt.Println(namespace)
	}

	return nil
}

func init() {
	RootCmd.AddCommand(namespacesCmd)
}
