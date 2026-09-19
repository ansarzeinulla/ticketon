#!/usr/bin/env bash
# BF-31 Move the create form to the client
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-28 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-31 S4 week-03-alinur-BF-31 origin/week-03-abylay-BF-28 'BF-31 Move the create form to the client'
