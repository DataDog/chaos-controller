package helpers

import (
	"golang.org/x/sys/unix"
)

var (
	Map32bit          MmapFlagArgument = MmapFlagArgument{rawValue: unix.MAP_32BIT, stringValue: "MAP_32BIT"}
)

func init() {
	mmapFlagMap[Map32bit.Value()] = Map32bit
}
