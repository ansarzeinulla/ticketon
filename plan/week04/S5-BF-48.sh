#!/usr/bin/env bash
# BF-48 Prove checkout cannot oversell
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-48 S5 week-04-olzhas-BF-48 origin/week-04 'BF-48 Prove checkout cannot oversell'
