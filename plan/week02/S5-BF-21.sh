#!/usr/bin/env bash
# BF-21 Add the Phase 1 specification
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-21 S5 week-02-olzhas-BF-21 origin/week-02 'BF-21 Add the Phase 1 specification'
