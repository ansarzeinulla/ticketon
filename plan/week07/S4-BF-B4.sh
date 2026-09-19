#!/usr/bin/env bash
# BF-B4 Read the GA4 id from a server-safe module
# Запускает Alinur Burlybayev (S4) сам, с этого компьютера, из любой папки.
# Строится поверх BF-75 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B4 S4 week-07-alinur-BF-B4 origin/week-07-alinur-BF-75 'BF-B4 Read the GA4 id from a server-safe module'
