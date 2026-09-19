#!/usr/bin/env bash
# BF-65 Add the admin portal
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-63 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-65 S4 week-06-alinur-BF-65 origin/week-06-abylay-BF-63 'BF-65 Add the admin portal'
