#!/usr/bin/env bash
# BF-B3 Rename middleware to proxy for Next 16
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B2 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B3 S4 week-06-alinur-BF-B3 origin/week-06-alinur-BF-B2 'BF-B3 Rename middleware to proxy for Next 16'
