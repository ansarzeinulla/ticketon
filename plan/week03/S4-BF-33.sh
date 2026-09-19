#!/usr/bin/env bash
# BF-33 List orders and attendees
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-30 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-33 S4 week-03-alinur-BF-33 origin/week-03-abylay-BF-30 'BF-33 List orders and attendees'
