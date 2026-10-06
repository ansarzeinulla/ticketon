#!/usr/bin/env bash
# BF-11 Add the signed-in shell
# Студент S4. Запускать с этого компьютера из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-11 S4 phase-01-alinur-BF-11 origin/phase-01-abylay-BF-10 "BF-11 Add the signed-in shell"
