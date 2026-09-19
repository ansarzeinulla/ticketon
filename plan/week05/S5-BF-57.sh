#!/usr/bin/env bash
# BF-57 Document how to run the scanner
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-56 (Olzhas Nurseit): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-57 S5 week-05-olzhas-BF-57 origin/week-05-olzhas-BF-56 'BF-57 Document how to run the scanner'
