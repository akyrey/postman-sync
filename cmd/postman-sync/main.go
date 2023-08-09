package main

import (
	"fmt"

	"github.com/akyrey/postman-sync/internal/postman"
)

func main() {
	pm := postman.Postman{ApiKey: ""}

    ws, err := pm.RetrieveWorkspaces()
    if err != nil {
        panic(err)
    }

    // ws, err := pm.RetrieveWorkspace("95e77d0a-45ef-4af4-842e-28855cfce6f6")
    // if err != nil {
    //     panic(err)
    // }
    // pm.CreateWorkspace(postman.WorkspaceBody{ Name: "Testing", Description: "Some description", Type: "personal" })
    // pm.DeleteWorkspace("a5339546-4363-49df-8580-e8cd25db3b51")
    fmt.Println(ws)
}
