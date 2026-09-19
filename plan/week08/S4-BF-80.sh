#!/usr/bin/env bash
# BF-80 Show reserved stock
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-79 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-80 S4 week-08-alinur-BF-80 origin/week-08-abylay-BF-79 'BF-80 Show reserved stock'
