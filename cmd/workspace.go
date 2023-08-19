package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	ErrMissingUid   error = errors.New("uid argument missing")
	ErrFolderExists error = errors.New("folder already exists")
)

func init() {
	rootCmd.AddCommand(workspaceCmd)

	workspaceCmd.AddCommand(lsCmd)
	workspaceCmd.AddCommand(getCmd)
	workspaceCmd.AddCommand(rmCmd)
	workspaceCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().StringVar(&wsCloneDest, "dest", "", "destination to save workspace info to (default is $HOME/.postman-sync/$workspaceUid)")
	cloneCmd.Flags().BoolVarP(&wsForceClone, "force", "f", false, "overwrite existing workspace folder if it exists")
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

	// Create destination folder
	err = os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		log.Fatalf("Unable to create destination directory: %v", err)
		return err
	}

	// Retrieve and save workspace
	ws, err := pm.RetrieveWorkspace(uid)
	if err != nil {
		log.Fatalf("Failed retrieving workspace: %v", err)
		return err
	}

	err = saveDataToJsonFile(ws, dest, uid)
	if err != nil {
		log.Fatalf("Unable to save workspace: %v", err)
		return err
	}

	// Retrieve and save collections
	collectionsFolder := fmt.Sprintf("%scollections/", dest)
	err = os.Mkdir(collectionsFolder, os.ModePerm)
	if err != nil {
		log.Fatalf("Unable to create collections directory: %v", err)
		return err
	}
	for i := range ws.Collections {
		cUid := ws.Collections[i].UID
		collection, err := pm.RetrieveCollection(cUid)
		if err != nil {
			log.Fatalf("Failed retrieving collection '%s': %v", cUid, err)
			return err
		}

		err = saveDataToJsonFile(collection, collectionsFolder, cUid)
		if err != nil {
			log.Fatalf("Unable to save collection: %v", err)
			return err
		}
	}

	// Retrieve and save environments
	environmentsFolder := fmt.Sprintf("%senvironments/", dest)
	err = os.Mkdir(environmentsFolder, os.ModePerm)
	if err != nil {
		log.Fatalf("Unable to create environments directory: %v", err)
		return err
	}
	for i := range ws.Environments {
		eUid := ws.Environments[i].UID
		environment, err := pm.RetrieveEnvironment(eUid)
		if err != nil {
			log.Fatalf("Failed retrieving environment '%s': %v", eUid, err)
			return err
		}

		err = saveDataToJsonFile(environment, environmentsFolder, eUid)
		if err != nil {
			log.Fatalf("Unable to save environment: %v", err)
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

func saveDataToJsonFile(data interface{}, dest, name string) error {
	file, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Fatalf("Unable to marshal data: %v", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("%s%s.json", dest, name), file, 0644)
	if err != nil {
		log.Fatalf("Unable to write data to file: %v", err)
		return err
	}

	return nil
}
