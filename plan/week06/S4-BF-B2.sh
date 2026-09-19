#!/usr/bin/env bash
# BF-B2 Move the session into an httpOnly cookie
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-64 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B2 S4 week-06-alinur-BF-B2 origin/week-06-abylay-BF-64 'BF-B2 Move the session into an httpOnly cookie'
