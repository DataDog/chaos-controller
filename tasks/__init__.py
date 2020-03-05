# Unless explicitly stated otherwise all files in this repository are licensed
# under the Apache License Version 2.0.
# This product includes software developed at Datadog (https://www.datadoghq.com/).
# Copyright 2020 Datadog, Inc.

"""
Invoke entrypoint, import here all the tasks we want to make available
"""
from invoke import  Collection
from .header import header_check

ns = Collection()

ns.add_task(header_check)
