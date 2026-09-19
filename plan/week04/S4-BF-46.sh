#!/usr/bin/env bash
# BF-46 Add the activation checklist
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-45 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-46 S4 week-04-alinur-BF-46 origin/week-04-abylay-BF-45 'BF-46 Add the activation checklist'
