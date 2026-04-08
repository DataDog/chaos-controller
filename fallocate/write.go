// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package fallocate

import (
	"io"
	"os"
)

const writeChunkSize = 65536

// fallocateWrite allocates disk space by writing zeros in chunks.
// Used as a fallback when the platform or filesystem doesn't support fallocate.
func fallocateWrite(file *os.File, offset int64, length int64) error {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	var buf [writeChunkSize]byte

	for length > 0 {
		n := int64(writeChunkSize)
		if length < n {
			n = length
		}

		if _, err := file.Write(buf[:n]); err != nil {
			return err
		}

		length -= n
	}

	return nil
}
