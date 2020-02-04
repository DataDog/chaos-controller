# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2020 Datadog, Inc.

#!/usr/bin/env python3

import csv
import spdx_lookup as lookup
import os
import glob

# read deps from go.sum file
deps = {}
with open('vendor/modules.txt', 'r') as modules:
    parent = ''
    for line in modules.readlines():
        if line.startswith('#'):
            parent = line.split()[1]
        else:
            if parent not in deps:
                deps[parent] = []
            deps[parent].append(line.strip())

# write deps to csv
with open('LICENSE-3rdparty.csv', 'w', newline='') as csvfile:
    w = csv.DictWriter(csvfile, ['From', 'Package', 'License'])
    w.writeheader()
    for parent, packages in deps.items():
        # find license file
        base_path = 'vendor/{}'.format(parent)
        license_filename = None
        if os.path.isfile('{}/LICENSE'.format(base_path)):
            license_filename = '{}/LICENSE'.format(base_path)
        else:
            license_files = glob.glob('{}/LICENSE.*'.format(base_path))
            if len(license_files) > 1:
                print('error: multiple license files for package {}'.format(parent))
            elif len(license_files) == 0:
                print('error: could not find license file for package {}'.format(parent))
            else:
                license_filename = license_files[0]

        # determine license type
        license_type = 'Unknown'
        if license_filename is not None:
            with open('{}'.format(license_filename), 'r') as license_file:
                match = lookup.match(license_file.read())
                if match is None:
                    print('error: could not determine license type for package {}'.format(parent))
                else:
                    license_type = match.license

        # write rows for child packages
        for package in packages:
                w.writerow({
                    'From': parent,
                    'Package': package,
                    'License': license_type,
                })
