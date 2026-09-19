#!/usr/bin/env bash
# BF-40 Add seat inventory
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-38 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-40 S2 week-04-alibi-BF-40 origin/week-04-ansar-BF-38 'BF-40 Add seat inventory'
