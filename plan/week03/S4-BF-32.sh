#!/usr/bin/env bash
# BF-32 Add the organizer dashboard
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-31 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-32 S4 week-03-alinur-BF-32 origin/week-03-alinur-BF-31 'BF-32 Add the organizer dashboard'
