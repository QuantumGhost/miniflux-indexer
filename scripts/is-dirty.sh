#!/usr/bin/env bash

# check if the current git repository is dirty (has modified / uncommitted files)
# if it's dirty, output ` (dirty)`
# else output nothing.

[[ -z $(git status --porcelain) ]] || echo ' (dirty)'
