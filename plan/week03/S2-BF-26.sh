#!/usr/bin/env bash
# BF-26 Manage ticket types
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-25 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-26 S2 week-03-alibi-BF-26 origin/week-03-alibi-BF-25 'BF-26 Manage ticket types'
