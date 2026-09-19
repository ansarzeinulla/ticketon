#!/usr/bin/env bash
# BF-74 Compute analytics from operational rows
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-73 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-74 S4 week-07-alinur-BF-74 origin/week-07-abylay-BF-73 'BF-74 Compute analytics from operational rows'
