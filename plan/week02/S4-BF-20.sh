#!/usr/bin/env bash
# BF-20 Keep the session in the browser
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-18 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-20 S4 week-02-alinur-BF-20 origin/week-02-abylay-BF-18 'BF-20 Keep the session in the browser'
