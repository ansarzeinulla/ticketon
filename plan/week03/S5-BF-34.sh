#!/usr/bin/env bash
# BF-34 Add the Phase 2 specification
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-34 S5 week-03-olzhas-BF-34 origin/week-03 'BF-34 Add the Phase 2 specification'
