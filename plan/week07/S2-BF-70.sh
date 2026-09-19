#!/usr/bin/env bash
# BF-70 Apply and cap discounts atomically
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-69 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-70 S2 week-07-alibi-BF-70 origin/week-07-alibi-BF-69 'BF-70 Apply and cap discounts atomically'
