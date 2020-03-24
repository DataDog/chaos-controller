#!/bin/bash

. $(dirname $0)/common
cmd=${@:2}
exec_into_pod "$1" "$cmd"
