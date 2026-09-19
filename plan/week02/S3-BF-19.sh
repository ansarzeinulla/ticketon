#!/usr/bin/env bash
# BF-19 Add reset and verify pages
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-20 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-19 S3 week-02-abylay-BF-19 origin/week-02-alinur-BF-20 'BF-19 Add reset and verify pages'
