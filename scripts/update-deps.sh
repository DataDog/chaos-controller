#!/usr/bin/env bash
set -euo pipefail

echo "Updating Python dependencies..."
pip install -q uv
uv pip compile --python-platform linux tasks/requirements.in -o tasks/requirements.txt
echo "Updated tasks/requirements.txt"
echo "Please commit both tasks/requirements.in and tasks/requirements.txt"
