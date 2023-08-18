package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var (
	workspaceCmd = &cobra.Command{
		Use:   "workspace",
		Short: "Manage Postman workspaces",
		Long:  `Allow CRUD operations via CLI to Postman workspaces`,
		RunE:  ls,
	}
	lsCmd = &cobra.Command{
		Use:   "ls",
		Short: "List workspaces",
		Long:  `List current API Key available workspaces`,
		RunE:  ls,
	}
	getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get single workspace",
		Long: `Retrieve info related to single workspace
The workspace retrieved is the one with the same uuid as the argument passed`,
		RunE: get,
	}
	rmCmd = &cobra.Command{
		Use:   "rm",
		Short: "Delete single workspace",
		Long:  `Delete the workspace with the given uuid`,
		RunE:  rm,
	}

	ErrMissingUuid error = errors.New("uuid argument missing")
)

func init() {
	rootCmd.AddCommand(workspaceCmd)

	workspaceCmd.AddCommand(lsCmd)
	workspaceCmd.AddCommand(getCmd)
	workspaceCmd.AddCommand(rmCmd)
}

func ls(_ *cobra.Command, _ []string) error {
	ws, err := pm.RetrieveWorkspaces()
	fmt.Println(ws)
	return err
}

func get(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		log.Fatalln("Missing uuid argument for workspace get")
		return ErrMissingUuid
	}

	ws, err := pm.RetrieveWorkspace(args[0])
	fmt.Println(ws)
	return err
}

func rm(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		log.Fatalln("Missing uuid argument for workspace rm")
		return ErrMissingUuid
	}

	err := pm.DeleteWorkspace(args[0])
	return err
}
