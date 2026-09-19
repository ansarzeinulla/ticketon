#!/usr/bin/env bash
# BF-81 Localise the remaining dashboard screens
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-80 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-81 S4 week-08-alinur-BF-81 origin/week-08-alinur-BF-80 'BF-81 Localise the remaining dashboard screens'
