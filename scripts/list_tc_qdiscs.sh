#!/bin/bash

. $(dirname $0)/common
[ $# -ne 1 ] && usage
exec_into_pod "$1" "tc qdisc"
