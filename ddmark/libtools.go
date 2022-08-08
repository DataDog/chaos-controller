// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package ddmark

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
)

// thisLibPath defines the local path for a given go library
func thisLibPath(apiname string) string {
	// commonLibPath is "$GOPATH/src/ddmarktemp/*api-name*"
	commonLibPath := genCommonLibPath()
	folderPath := fmt.Sprintf("%v/%v/", commonLibPath, apiname)

	return folderPath
}

// genCommonLibPath defines the local path for all ddmark-copied libraries
func genCommonLibPath() string {
	return fmt.Sprintf("%v/%v/%v", os.Getenv("GOPATH"), "src", "ddmarktemp")
}

// initLibrary copies a binary-embedded API into a custom folder in GOPATH.
// This way, it can be read by ddmark.
func InitLibrary(embeddedFS embed.FS, apiname string) error {
	if _, isGoInstalled := os.LookupEnv("GOPATH"); !isGoInstalled {
		err := fmt.Errorf("ddmark lib setup error: please make sure go (1.18 or higher) is installed and the GOPATH is set")
		return err
	}

	if err := os.Setenv("GO111MODULE", "off"); err != nil {
		return fmt.Errorf("ddmark lib setup error: %w", err)
	}

	folderPath := thisLibPath(apiname)

	if err := os.MkdirAll(folderPath, 0750); err != nil {
		return fmt.Errorf("ddmark lib setup error: %w", err)
	}

	err := fs.WalkDir(embeddedFS, ".",
		// this function is executed for every file found within the binary-embedded folder
		// it copies every files to another location on the computer through io.Copy
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			fin, err := fs.ReadFile(embeddedFS, path)
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			fout, err := os.Create(folderPath + info.Name())
			if err != nil {
				return err
			}

			if _, err = fout.Write(fin); err != nil {
				return err
			}

			if err = fout.Close(); err != nil {
				return err
			}

			return nil
		})

	if err != nil {
		return fmt.Errorf("ddmark lib setup error: %w", err)
	}

	return nil
}

// CleanupLibraries deletes all libraries in the common ddmark lib folder ($GOPATH/src/ddmarktemp)
func CleanupLibraries() error {
	folderPath := genCommonLibPath()

	if os.RemoveAll(folderPath) != nil {
		return fmt.Errorf("ddmark: couldn't clean up API located at " + folderPath)
	}

	return nil
}
