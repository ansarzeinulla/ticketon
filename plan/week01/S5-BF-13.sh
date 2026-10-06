#!/usr/bin/env bash
# BF-13 Add CI
# Студент S5. Запускать с этого компьютера из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-13 S5 phase-01-olzhas-BF-13 origin/phase-01-olzhas-BF-12 "BF-13 Add CI"
