#!/usr/bin/env bash
# BF-52 Charge a processing fee
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-51 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-52 S2 week-05-alibi-BF-52 origin/week-05-alibi-BF-51 'BF-52 Charge a processing fee'
