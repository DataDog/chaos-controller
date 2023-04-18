// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package disk

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Informer represents a disk informer giving information about
// the device
type Informer interface {
	Major() int
	Source() string
}

type disk struct {
	major  int
	source string
}

// FromPath returns a disk informer from the given path
func FromPath(path string) (Informer, error) {
	// ensure the file exists before going further
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	// get host path device
	df := exec.Command("df", "--output=source", path)

	out, err := df.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing df command on %s: %w", path, err)
	}

	// parse df output
	lines := []string{}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// we expect 2 lines here, the first one being the header
	if len(lines) != 2 {
		return nil, fmt.Errorf("unexpected df output: %v", lines)
	}

	// get the second line (and get rid of the header)
	device := lines[1]

	// get device major identifier
	ls := exec.Command("ls", "-lH", device)

	out, err = ls.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error executing ls command: %w\noutput: %s", err, out)
	}

	// parse ls output, format is like:
	//	<permissions> <size> <user> <group> <major>, <minor> <month> <day> <time> <path>
	// example:
	//	# ls -lH /dev/sda1
	//	brw-rw---- 1 root disk 8, 1 Jul  7 09:51 /dev/sda1
	lsParsed := strings.Split(string(out), " ")
	if len(lsParsed) < 10 {
		return nil, fmt.Errorf("unexpected ls output: %s", string(out))
	}

	// 4th field is the major number with a coma (cf. ls format above)
	// example: 8,
	majorS := strings.TrimSuffix(lsParsed[4], ",")

	// cast major identifier to a number
	major, err := strconv.Atoi(majorS)
	if err != nil {
		return nil, fmt.Errorf("unexpected device major identifier (%s): %w", majorS, err)
	}

	return disk{
		major:  major,
		source: device,
	}, nil
}

func (d disk) Major() int {
	return d.major
}

func (d disk) Source() string {
	return d.source
}
