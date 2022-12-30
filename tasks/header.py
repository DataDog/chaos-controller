#!/usr/bin/env python3

# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2023 Datadog, Inc.

from invoke import task
import glob
import typing as t
import os

file_extension_map = {
    "go": "//",
    "yaml": "#",
    "yml": "#",
    "py": "#",
}

# fixed list of auto_generated headers
# don't forget to add newlines
auto_generated_headers = [
    "// +build !ignore_autogenerated\n",
    "#!/usr/bin/env python3\n",
]

files_to_skip = [
    "api/v1beta1/zz_generated.deepcopy.go",
    "bin/injector/dns_disruption_resolver.py",
    "chart/templates/crds/chaos.datadoghq.com_disruptions.yaml",
    "chart/templates/role.yaml",
    "chart/install.yaml",
    "cpuset/cpuset.go",
    "grpc/disruptionlistener/disruptionlistener_grpc.pb.go",
    "grpc/disruptionlistener/disruptionlistener.pb.go",
    "dogfood/chaosdogfood/chaosdogfood_grpc.pb.go",
    "dogfood/chaosdogfood/chaosdogfood.pb.go",
]

# generates the header with the given comment tag
def generate_header_with_tag(tag: str) -> t.List:
    return [
        f"{tag} Unless explicitly stated otherwise all files in this repository are licensed\n",
        f"{tag} under the Apache License Version 2.0.\n",
        f"{tag} This product includes software developed at Datadog (https://www.datadoghq.com/).\n",
        f"{tag} Copyright 2023 Datadog, Inc.\n",
    ]


# prints all files (with its relative path) having the given extension
def get_files_with_extension(extension: str) -> t.List:
    return [
        file
        for file in glob.glob(f"**/*.{extension}", recursive=True)
        if not file.startswith("vendor/")
    ]


# return the position of the header.
# on autogenerated files this can be different.
def get_header_position(file_name: str) -> t.Dict:
    header_position = {"starting_pos": 0, "ending_pos": 4}

    with open(file_name, "r") as f:
        first_line = f.readline()

    # if the first line is a auto generated one
    # change the header positions, currently hardcoded to +2
    if first_line in auto_generated_headers:
        header_position["starting_pos"] += 2
        header_position["ending_pos"] += 2

    return header_position


# returns true if the header generated with the given comment tag
# is present in the given file, false otherwise
def header_is_present(file: str, header: t.List, header_position: t.Dict) -> bool:
    with open(file, "r") as f:
        file_header = f.readlines()[
            header_position["starting_pos"] : header_position["ending_pos"]
        ]

    if file_header == header:
        return True

    return False


# update the header starting on given position
def update_header(file: str, header: t.List, header_position: t.Dict):
    with open(file, "r") as f:
        content = f.readlines()

    # incoming header is a list
    header = "".join(header)

    # reinsert the header on the given position
    content[header_position["starting_pos"]:header_position["ending_pos"]] = header
    content = "".join(content)

    with open(file, "w") as f:
        f.write(content)


@task
def header_check(ctx):
    """ Update headers for supported files """
    exit_code = 0
    for extension, tag in file_extension_map.items():
        print(f"dealing with {extension} files")
        header = generate_header_with_tag(tag)
        files = get_files_with_extension(extension)
        for file in files:
            if file in files_to_skip:
                print(f"skipping file {file}")
                continue
            header_position = get_header_position(file)
            if not header_is_present(file, header, header_position):
                print(f"header missing in {file}")
                update_header(file, header, header_position)
                exit_code = 1
    exit(exit_code)
