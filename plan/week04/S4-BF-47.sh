#!/usr/bin/env bash
# BF-47 Show ticket status per order
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-46 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-47 S4 week-04-alinur-BF-47 origin/week-04-alinur-BF-46 'BF-47 Show ticket status per order'
