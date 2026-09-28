package transpiler

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Resource payloads are copied under an underscore directory to prevent .go
// resources from being discovered as source by `go build ./...`.
func embedProjectResources(generated, module string) (string, error) {
	root := filepath.Join(generated, "resources")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		count++
		return writeProjectFile(filepath.Join(generated, "java2go_resources", "_data", rel), data)
	})
	if err != nil || count == 0 {
		return "", err
	}
	source := `package java2go_resources
import("embed";"io/fs";stdjava "github.com/NickyBoy89/java2go/stdjava")
//go:embed all:_data
var resources embed.FS
func init(){root,err:=fs.Sub(resources,"_data");if err!=nil{panic(err)};stdjava.RegisterClassResources(root)}
`
	if err = writeProjectFile(filepath.Join(generated, "java2go_resources", "resources.go"), []byte(source)); err != nil {
		return "", err
	}
	// Keep the historical on-disk resources copy available to applications,
	// while preventing resource payloads named *.go from entering ./... builds.
	// Add this build boundary only after embedding the original payloads, so it
	// never becomes a synthetic Java classpath resource. User go.mod is retained.
	boundary := filepath.Join(root, "go.mod")
	if _, statErr := os.Stat(boundary); os.IsNotExist(statErr) {
		if err = writeProjectFile(boundary, []byte("module java2go.resource.payloads\n")); err != nil {
			return "", err
		}
	} else if statErr != nil {
		return "", statErr
	}
	return fmt.Sprintf("_ %q", module+"/java2go_resources"), nil
}
