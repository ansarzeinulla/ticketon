#!/usr/bin/env bash
# BF-28 Add the formatting dependencies
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-27 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-28 S3 week-03-abylay-BF-28 origin/week-03-alibi-BF-27 'BF-28 Add the formatting dependencies'
