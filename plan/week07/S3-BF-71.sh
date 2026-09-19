#!/usr/bin/env bash
# BF-71 Add the English dictionary
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-70 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-71 S3 week-07-abylay-BF-71 origin/week-07-alibi-BF-70 'BF-71 Add the English dictionary'
