#!/usr/bin/env python3
"""Regenerate the table registry from explicit TableName receiver declarations.
Run from repository root; this does not generate or modify DAO code.
"""
from pathlib import Path
import re
entries, packages = [], {}
for source in sorted(Path("internal/model").rglob("*.go")):
    for name in re.findall(r"func\s+\(\s*\*?(\w+)\s*\)\s+TableName\(\)", source.read_text()):
        alias = packages.setdefault(source.parent, "m" + str(len(packages)))
        entries.append((alias, name, str(source)))
Path("internal/schema/models.go").write_text(
    "// Code generated from TableName declarations by cmd/schema/registry.py; DO NOT EDIT.\npackage schema\nimport (\n"
    + "".join(f"{alias} \"github.com/wxlbd/ruoyi-mall-go/{path}\"\n" for path, alias in packages.items())
    + ")\ntype ModelSource struct { Model any; Source string }\nvar Models = []ModelSource{\n"
    + "".join(f"{{&{alias}.{name}{{}}, \"{path}\"}},\n" for alias, name, path in entries)
    + "}\n"
)
