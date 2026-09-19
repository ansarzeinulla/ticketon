#!/usr/bin/env bash
# BF-B10 Name the check-in reversal conflict
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B8 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B10 S5 week-08-olzhas-BF-B10 origin/week-08-alibi-BF-B8 'BF-B10 Name the check-in reversal conflict'
