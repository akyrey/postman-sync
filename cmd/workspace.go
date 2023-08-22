package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"

	"github.com/akyrey/postman-sync/pkg/postman"
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
The workspace retrieved is the one with the same uid as the argument passed`,
		RunE: get,
	}
	rmCmd = &cobra.Command{
		Use:   "rm",
		Short: "Delete single workspace",
		Long:  `Delete the workspace with the given uid`,
		RunE:  rm,
	}
	wsCloneDest  string
	wsForceClone bool
	cloneCmd     = &cobra.Command{
		Use:   "clone",
		Short: "Clone the given workspace",
		Long: `Clone the given workspace.
Retrieve the workspace related to the uid passed as argument and save it inside the configured folder
All its collections and environments (currently mocks and APIs aren't supported) are saved too inside the relative folders`,
		RunE: clone,
	}
	wsPushSource string
	pushCmd      = &cobra.Command{
		Use:   "push",
		Short: "Push the given workspace",
		Long: `Push the given workspace.
Find the workspace with the given uid inside the configured folder, and create it anew inside user api-key Postman
All its collections and environments (currently mocks and APIs aren't supported) are created as well`,
		RunE: push,
	}

	ErrMissingUid        error = errors.New("uid argument missing")
	ErrFolderExists      error = errors.New("folder already exists")
	ErrFolderDoesntExist error = errors.New("folder doesn't exist")
)

func init() {
	rootCmd.AddCommand(workspaceCmd)

	workspaceCmd.AddCommand(lsCmd)
	workspaceCmd.AddCommand(getCmd)
	workspaceCmd.AddCommand(rmCmd)
	workspaceCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().StringVar(&wsCloneDest, "dest", "", "destination to save workspace info to (default is $HOME/.postman-sync/$workspaceUid)")
	cloneCmd.Flags().BoolVarP(&wsForceClone, "force", "f", false, "overwrite existing workspace folder if it exists")
	workspaceCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringVar(&wsPushSource, "source", "", "source to load workspace info from (default is $HOME/.postman-sync/$workspaceUid)")
}

func ls(_ *cobra.Command, _ []string) error {
	ws, err := pm.RetrieveWorkspaces()
	fmt.Println(ws)
	return err
}

func get(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		slog.Error("Missing uid argument for workspace get")
		return ErrMissingUid
	}

	ws, err := pm.RetrieveWorkspace(args[0])
	fmt.Println(ws)
	return err
}

func rm(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		slog.Error("Missing uid argument for workspace rm")
		return ErrMissingUid
	}

	err := pm.DeleteWorkspace(args[0])
	return err
}

func clone(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		slog.Error("Missing uid argument for workspace pull")
		return ErrMissingUid
	}

	uid := args[0]
	dest := getDestination(uid)

	_, err := os.Stat(dest)
	if !wsForceClone && !os.IsNotExist(err) {
		slog.Error("Folder already exists and `force` not provided", dest)
		return ErrFolderExists
	}

	// Delete folder if it exists
	if wsForceClone && !os.IsNotExist(err) {
		err = os.RemoveAll(dest)
		if err != nil {
			slog.Error("Unable to remove existing workspace folder: ", err)
			return err
		}
	}

	// Create destination folder
	err = os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		slog.Error("Unable to create destination directory: ", err)
		return err
	}

	// Retrieve and save workspace
	ws, err := pm.RetrieveWorkspace(uid)
	if err != nil {
		slog.Error("Failed retrieving workspace: ", err)
		return err
	}

	err = saveDataToJsonFile(ws, dest, uid)
	if err != nil {
		slog.Error("Unable to save workspace: ", err)
		return err
	}

	// Retrieve and save collections
	collectionsFolder := fmt.Sprintf("%scollections/", dest)
	err = os.Mkdir(collectionsFolder, os.ModePerm)
	if err != nil {
		slog.Error("Unable to create collections directory: ", err)
		return err
	}
	for i := range ws.Collections {
		cUid := ws.Collections[i].UID
		collection, err := pm.RetrieveCollection(cUid)
		if err != nil {
			slog.Error("Failed retrieving collection", cUid, err)
			return err
		}

		err = saveDataToJsonFile(collection, collectionsFolder, cUid)
		if err != nil {
			slog.Error("Unable to save collection: ", err)
			return err
		}
	}

	// Retrieve and save environments
	environmentsFolder := fmt.Sprintf("%senvironments/", dest)
	err = os.Mkdir(environmentsFolder, os.ModePerm)
	if err != nil {
		slog.Error("Unable to create environments directory: ", err)
		return err
	}
	for i := range ws.Environments {
		eUid := ws.Environments[i].UID
		environment, err := pm.RetrieveEnvironment(eUid)
		if err != nil {
			slog.Error("Failed retrieving environment", eUid, err)
			return err
		}

		err = saveDataToJsonFile(environment, environmentsFolder, eUid)
		if err != nil {
			slog.Error("Unable to save environment: ", err)
			return err
		}
	}

	return nil
}

func push(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		slog.Error("Missing uid argument for workspace pull")
		return ErrMissingUid
	}

	uid := args[0]
	source := getSource(uid)

	if _, err := os.Stat(source); os.IsNotExist(err) {
		slog.Error("Folder doesn't exist", source)
		return ErrFolderDoesntExist
	}

	// Read workspace data and create it
	workspaceFile, err := os.ReadFile(fmt.Sprintf("%s%s.json", source, uid))
	if err != nil {
		slog.Error("Unable to read data from file: ", err)
		return err
	}
	var workspace postman.Workspace
	err = json.Unmarshal(workspaceFile, &workspace)
	if err != nil {
		slog.Error("Unable to unmarshal workspace data: ", err)
		return err
	}
	ws, err := pm.CreateWorkspace(postman.CreateWorkspaceParam{Name: workspace.Name, Description: workspace.Description, Type: workspace.Type})
	if err != nil {
		slog.Error("Unable to create workspace: ", err)
		return err
	}
	uid = ws.ID

	// Read all collections data and create each
	collectionsFolder := fmt.Sprintf("%scollections/", source)
	if _, err := os.Stat(collectionsFolder); os.IsNotExist(err) {
		slog.Error("Folder doesn't exist", collectionsFolder)
		return ErrFolderDoesntExist
	}
	for i := range workspace.Collections {
		cUid := workspace.Collections[i].UID
		collectionFile, err := os.ReadFile(fmt.Sprintf("%s%s.json", collectionsFolder, cUid))
		if err != nil {
			slog.Error("Unable to read data from file: ", err)
			return err
		}
		var collection postman.CreateCollectionParam
		err = json.Unmarshal(collectionFile, &collection)
		if err != nil {
			slog.Error("Unable to unmarshal collection data: ", err)
			return err
		}
		c, err := pm.CreateCollection(collection, uid)
		if err != nil {
			slog.Error("Unable to create collection: ", err)
			return err
		}
		slog.Debug("Created collection", c)
	}

	// Read all environments data and create each
	environmentsFolder := fmt.Sprintf("%senvironments/", source)
	if _, err := os.Stat(environmentsFolder); os.IsNotExist(err) {
		slog.Error("Folder doesn't exist", environmentsFolder)
		return ErrFolderDoesntExist
	}
	for i := range workspace.Environments {
		eUid := workspace.Environments[i].UID
		environmentFile, err := os.ReadFile(fmt.Sprintf("%s%s.json", environmentsFolder, eUid))
		if err != nil {
			slog.Error("Unable to read data from file: ", err)
			return err
		}
		var environment postman.CreateEnvironmentParam
		err = json.Unmarshal(environmentFile, &environment)
		if err != nil {
			slog.Error("Unable to unmarshal environment data: ", err)
			return err
		}
		e, err := pm.CreateEnvironment(environment, uid)
		if err != nil {
			slog.Error("Unable to create environment: ", err)
			return err
		}
		slog.Debug("Created environment", e)
	}

	return nil
}

// Retrieve clone destination to use
// First checks `dest` flag, then `workspace.clone.dest` config value and fallback to `$HOME/.postman-sync/$uid`
func getDestination(uid string) string {
	if wsCloneDest != "" {
		return wsCloneDest
	}

	if viper.IsSet("workspace.clone.dest") {
		return viper.GetString("workspace.clone.dest")
	}

	return fmt.Sprintf("/home/akyrey/.postman-sync/%s/", uid)
}

// Retrieve push source to use
// First checks `source` flag, then `workspace.push.source` config value and fallback to `$HOME/.postman-sync/$uid`
func getSource(uid string) string {
	if wsPushSource != "" {
		return wsPushSource
	}

	if viper.IsSet("workspace.push.source") {
		return viper.GetString("workspace.push.source")
	}

	return fmt.Sprintf("/home/akyrey/.postman-sync/%s/", uid)
}

func saveDataToJsonFile(data interface{}, dest, name string) error {
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		slog.Error("Unable to marshal data: ", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("%s%s.json", dest, name), file, 0644)
	if err != nil {
		slog.Error("Unable to write data to file: ", err)
		return err
	}

	return nil
}
