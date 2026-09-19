#!/usr/bin/env bash
# BF-43 Show the activation notice
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-42 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-43 S3 week-04-abylay-BF-43 origin/week-04-alibi-BF-42 'BF-43 Show the activation notice'
