#!/usr/bin/env bash
# BF-35 Add the Phase 3 specification
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-35 S5 week-03-olzhas-BF-35 origin/week-03 'BF-35 Add the Phase 3 specification'
