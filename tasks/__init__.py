# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2024 Datadog, Inc.

from invoke import  Collection

from .header import header_check
from .thirdparty import license_check

ns = Collection()

ns.add_task(header_check)
ns.add_task(license_check)
