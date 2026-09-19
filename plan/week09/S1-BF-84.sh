#!/usr/bin/env bash
# BF-84 Say required instead of too short
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-84 S1 week-09-ansar-BF-84 origin/week-09 'BF-84 Say required instead of too short'
