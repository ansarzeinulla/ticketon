#!/usr/bin/env bash
# BF-50 Update the money assertions for the fee
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-52 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-50 S1 week-05-ansar-BF-50 origin/week-05-alibi-BF-52 'BF-50 Update the money assertions for the fee'
