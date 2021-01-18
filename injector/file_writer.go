// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector

import "os"

// FileWriter is a component allowing to write the given data
// to the given file
type FileWriter interface {
	Write(path string, mode os.FileMode, data string) error
}

// standardFileWriter implements the FileWriter interface
type standardFileWriter struct {
	dryRun bool
}

// Write writes the given data to the given file
func (fw standardFileWriter) Write(path string, mode os.FileMode, data string) error {
	// early exit if dry-run mode is enabled
	if fw.dryRun {
		return nil
	}

	f, err := os.OpenFile(path, os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	_, err = f.WriteString(data)
	if err != nil {
		return err
	}

	return f.Close()
}
