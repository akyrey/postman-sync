package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
		log.Fatalln("Missing uid argument for workspace get")
		return ErrMissingUid
	}

	ws, err := pm.RetrieveWorkspace(args[0])
	fmt.Println(ws)
	return err
}

func rm(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		log.Fatalln("Missing uid argument for workspace rm")
		return ErrMissingUid
	}

	err := pm.DeleteWorkspace(args[0])
	return err
}

func clone(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		log.Fatalln("Missing uid argument for workspace pull")
		return ErrMissingUid
	}

	uid := args[0]
	dest := getDestination(uid)

	_, err := os.Stat(dest)
	if !wsForceClone && !os.IsNotExist(err) {
		log.Fatalf("Folder %s already exists and `force` not provided\n", dest)
		return ErrFolderExists
	}

	// Delete folder if it exists
	if wsForceClone && !os.IsNotExist(err) {
		err = os.RemoveAll(dest)
		if err != nil {
			log.Fatalf("Unable to remove existing workspace folder: %v\n", err)
			return err
		}
	}

	// Create destination folder
	err = os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		log.Fatalf("Unable to create destination directory: %v\n", err)
		return err
	}

	// Retrieve and save workspace
	ws, err := pm.RetrieveWorkspace(uid)
	if err != nil {
		log.Fatalf("Failed retrieving workspace: %v\n", err)
		return err
	}

	err = saveDataToJsonFile(ws, dest, uid)
	if err != nil {
		log.Fatalf("Unable to save workspace: %v\n", err)
		return err
	}

	// Retrieve and save collections
	collectionsFolder := fmt.Sprintf("%scollections/", dest)
	err = os.Mkdir(collectionsFolder, os.ModePerm)
	if err != nil {
		log.Fatalf("Unable to create collections directory: %v\n", err)
		return err
	}
	for i := range ws.Collections {
		cUid := ws.Collections[i].UID
		collection, err := pm.RetrieveCollection(cUid)
		if err != nil {
			log.Fatalf("Failed retrieving collection '%s': %v\n", cUid, err)
			return err
		}

		err = saveDataToJsonFile(collection, collectionsFolder, cUid)
		if err != nil {
			log.Fatalf("Unable to save collection: %v\n", err)
			return err
		}
	}

	// Retrieve and save environments
	environmentsFolder := fmt.Sprintf("%senvironments/", dest)
	err = os.Mkdir(environmentsFolder, os.ModePerm)
	if err != nil {
		log.Fatalf("Unable to create environments directory: %v\n", err)
		return err
	}
	for i := range ws.Environments {
		eUid := ws.Environments[i].UID
		environment, err := pm.RetrieveEnvironment(eUid)
		if err != nil {
			log.Fatalf("Failed retrieving environment '%s': %v\n", eUid, err)
			return err
		}

		err = saveDataToJsonFile(environment, environmentsFolder, eUid)
		if err != nil {
			log.Fatalf("Unable to save environment: %v\n", err)
			return err
		}
	}

	return nil
}

func push(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		log.Fatalln("Missing uid argument for workspace pull")
		return ErrMissingUid
	}

	uid := args[0]
	source := getSource(uid)

	if _, err := os.Stat(source); os.IsNotExist(err) {
		log.Fatalf("Folder %s doesn't exist\n", source)
		return ErrFolderDoesntExist
	}

	// Read workspace data and create it
	workspaceFile, err := os.ReadFile(fmt.Sprintf("%s%s.json", source, uid))
	if err != nil {
		log.Fatalf("Unable to read data from file: %v\n", err)
		return err
	}
	var workspace postman.Workspace
	err = json.Unmarshal(workspaceFile, &workspace)
	if err != nil {
		log.Fatalf("Unable to unmarshal workspace data: %v\n", err)
		return err
	}
	ws, err := pm.CreateWorkspace(workspace)
	if err != nil {
		log.Fatalf("Unable to create workspace: %s\n", err)
		return err
	}
	uid = ws.ID

	// Read all collections data and create each
	collectionsFolder := fmt.Sprintf("%scollections/", source)
	if _, err := os.Stat(collectionsFolder); os.IsNotExist(err) {
		log.Fatalf("Folder %s doesn't exist\n", collectionsFolder)
		return ErrFolderDoesntExist
	}
	for i := range workspace.Collections {
		cUid := workspace.Collections[i].UID
		collectionFile, err := os.ReadFile(fmt.Sprintf("%s%s.json", collectionsFolder, cUid))
		if err != nil {
			log.Fatalf("Unable to read data from file: %v\n", err)
			return err
		}
		var collection postman.Collection
		err = json.Unmarshal(collectionFile, &collection)
		if err != nil {
			log.Fatalf("Unable to unmarshal collection data: %v\n", err)
			return err
		}
		_, err = pm.CreateCollection(collection, uid)
		if err != nil {
			log.Fatalf("Unable to create collection: %s\n", err)
			return err
		}
	}

	// Read all environments data and create each
	environmentsFolder := fmt.Sprintf("%senvironments/", source)
	if _, err := os.Stat(environmentsFolder); os.IsNotExist(err) {
		log.Fatalf("Folder %s doesn't exist\n", environmentsFolder)
		return ErrFolderDoesntExist
	}
	for i := range workspace.Environments {
		eUid := workspace.Environments[i].UID
		environmentFile, err := os.ReadFile(fmt.Sprintf("%s%s.json", environmentsFolder, eUid))
		if err != nil {
			log.Fatalf("Unable to read data from file: %v\n", err)
			return err
		}
		var environment postman.Environment
		err = json.Unmarshal(environmentFile, &environment)
		if err != nil {
			log.Fatalf("Unable to unmarshal environment data: %v\n", err)
			return err
		}
		_, err = pm.CreateEnvironment(environment, uid)
		if err != nil {
			log.Fatalf("Unable to create environment: %s\n", err)
			return err
		}
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
		log.Fatalf("Unable to marshal data: %v\n", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("%s%s.json", dest, name), file, 0644)
	if err != nil {
		log.Fatalf("Unable to write data to file: %v\n", err)
		return err
	}

	return nil
}
