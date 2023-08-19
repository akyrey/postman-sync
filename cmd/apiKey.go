package cmd

import (
	"fmt"
	"syscall"

	kr "github.com/99designs/keyring"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/akyrey/postman-sync/pkg/keyring"
)

var (
	apiKeyCmd = &cobra.Command{
		Use:   "apiKey",
		Short: "Manage API Keys",
		Long:  `Allow setting a new API Key`,
		RunE:  apiKey,
	}
	forceApiKey  bool
	setCmd = &cobra.Command{
		Use:   "set",
		Short: "Set API Key",
		Long: `Store API key inside local key store.
This key will be used for all requests to Postman APIs`,
		RunE: set,
	}
)

func init() {
	rootCmd.AddCommand(apiKeyCmd)

	apiKeyCmd.AddCommand(setCmd)
	setCmd.Flags().BoolVarP(&forceApiKey, "force", "f", false, "Overwrite existing stored API Key")
}

func apiKey(_ *cobra.Command, _ []string) error {
	_, err := keyring.Store.Get("api-key")
	if err != nil {
		fmt.Println("Unable to find API Key in local key store. Use `postman-sync apiKey set` to set one")
		return err
	}

	fmt.Println("Found an API Key inside local key store")
	return nil
}

func set(_ *cobra.Command, _ []string) error {
	if _, err := keyring.Store.Get("api-key"); !forceApiKey && err == nil {
		fmt.Println("API Key already stored, use --force to overwrite")
		return nil
	}

	fmt.Print("Please enter your API Key: ")
	apiKeyByte, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("Unable to read API Key %v", err)
		return err
	}

	err = keyring.Store.Set(kr.Item{Key: "api-key", Label: "API Key", Description: "Postman API Key used for every request", Data: apiKeyByte})
	if err != nil {
		fmt.Printf("Unable to store API Key: %v", err)
		return err
	}

	fmt.Println("\nSuccessfully stored API Key")
	return nil
}
