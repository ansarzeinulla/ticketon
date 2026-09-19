#!/usr/bin/env bash
# BF-B8 Name the refund conflict
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-78 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B8 S2 week-08-alibi-BF-B8 origin/week-08-alibi-BF-78 'BF-B8 Name the refund conflict'
