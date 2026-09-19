#!/usr/bin/env bash
# BF-75 Localise the dashboard
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-74 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-75 S4 week-07-alinur-BF-75 origin/week-07-alinur-BF-74 'BF-75 Localise the dashboard'
