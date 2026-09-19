#!/usr/bin/env bash
# BF-63 Add the support widget to the order page
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B3 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-63 S3 week-06-abylay-BF-63 origin/week-06-alinur-BF-B3 'BF-63 Add the support widget to the order page'
