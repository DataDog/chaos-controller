#!/usr/local/bin/bash

# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2020 Datadog, Inc.

# generates the header with the given comment tag
function generate_header_with_tag(){
  cat <<EOI
$1 Unless explicitly stated otherwise all files in this repository are licensed
$1 under the Apache License Version 2.0.
$1 This product includes software developed at Datadog (https://www.datadoghq.com/).
$1 Copyright 2020 Datadog, Inc.
EOI
}

# prints all files (with its relative path) having the given extension
function get_files_with_extension(){
  find ./ -iname "*.$1" -not -path '*/vendor/*' | sed 's#^.//##'
}

# inserts the header generated with the given comment tag
# at the very beginning of the given file
function insert_header_with_tag(){
  printf '%s\n\n%s\n' "$(generate_header_with_tag $2)" "$(cat $1)" >$1
}

# returns true if the header generated with the given comment tag
# is present in the given file, false otherwise
function header_is_present(){
  header=$(head -n4 $1)
  expected=$(generate_header_with_tag $2)
  [ "$header" == "$expected" ]
}

# declare extensions and associated comment tag
declare -A EXTS
EXTS["go"]="//"
EXTS["yaml"]="#"
EXTS["yml"]="#"

# insert header if not already present
exit_code=0
for ext in "${!EXTS[@]}"; do
  echo "dealing with $ext files"
  tag=${EXTS[$ext]}
  files=$(get_files_with_extension $ext)
  for file in $files; do
    if ! header_is_present $file $tag; then
      echo "header is missing in $file"
      exit_code=1
      insert_header_with_tag $file $tag
    fi
  done
done

exit $exit_code
