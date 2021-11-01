#!/usr/bin/env python3

# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2021 Datadog, Inc.

import csv
import glob
import os
import sys

from invoke import task
import spdx_lookup as lookup

csv_file = 'LICENSE-3rdparty.csv'

@task
def license_check(ctx):
    """ Build LICENSE-3rdparty.csv """
    # read deps from go.sum file
    deps = {}
    with open('vendor/modules.txt', 'r') as modules:
        parent = ''
        for line in modules.readlines():
            if line.startswith('##'):
                continue
            elif line.startswith('#'):
                parent = line.split()[1]
            else:
                if parent not in deps:
                    deps[parent] = []
                deps[parent].append(line.strip())

    # load existing csv for further checks
    existing_deps = {}
    with open(csv_file, 'r', newline='') as csvfile:
        reader = csv.reader(csvfile)
        for row in reader:
            existing_deps[row[0]] = row[2]

    # write deps to csv
    with open(csv_file, 'w', newline='') as csvfile:
        w = csv.DictWriter(csvfile, ['From', 'Package', 'License'])
        w.writeheader()
        for parent, packages in deps.items():
            error = None

            # find license file
            base_path = 'vendor/{}'.format(parent)
            license_filename = None
            if os.path.isfile('{}/LICENSE'.format(base_path)):
                license_filename = '{}/LICENSE'.format(base_path)
            else:
                license_files = glob.glob('{}/LICENSE.*'.format(base_path))
                if len(license_files) > 1:
                    error = 'error: multiple license files for package {}'.format(parent)
                elif len(license_files) == 0:
                    error = 'error: could not find license file for package {}'.format(parent)
                else:
                    license_filename = license_files[0]

            # determine license type
            license_type = 'Unknown'
            if license_filename is not None:
                with open('{}'.format(license_filename), 'r') as license_file:
                    match = lookup.match(license_file.read())
                    if match is None:
                        error = 'error: could not determine license type for package {}'.format(parent)
                    else:
                        license_type = match.license.id

            # if the license has already been defined in the CSV, multiple options:
            # - an error occured during recognition but the license has already been specified, we can keep it
            # - no error occured but the determined license is different than the one specified in the CSV (update needed)
            # if it hasn't been defined in the CSV yet but an error occured, just exit
            # otherwise, we're fine
            if parent in existing_deps:
                existing_license_type = existing_deps[parent]
                if error is not None:
                    print('warning: license type for package {} cannot be determined but has already been specified: {}'.format(
                        parent,
                        existing_license_type))
                    license_type = existing_license_type
                elif str(existing_license_type) != str(license_type):
                    print('warning: license may be outdated for package {} (actual: {}, detected: {})'.format(
                        parent,
                        existing_license_type,
                        license_type))
            elif error is not None:
                print(error)
                sys.exit(2)

            # write rows for child packages
            for package in packages:
                    w.writerow({
                        'From': parent,
                        'Package': package,
                        'License': license_type,
                    })
