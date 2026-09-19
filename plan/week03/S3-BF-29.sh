#!/usr/bin/env bash
# BF-29 Add the public catalogue
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-32 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-29 S3 week-03-abylay-BF-29 origin/week-03-alinur-BF-32 'BF-29 Add the public catalogue'
