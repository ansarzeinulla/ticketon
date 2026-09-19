#!/usr/bin/env bash
# BF-30 Add the order page
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-29 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-30 S3 week-03-abylay-BF-30 origin/week-03-abylay-BF-29 'BF-30 Add the order page'
