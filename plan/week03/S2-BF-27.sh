#!/usr/bin/env bash
# BF-27 Add the checkout transaction
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-26 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-27 S2 week-03-alibi-BF-27 origin/week-03-alibi-BF-26 'BF-27 Add the checkout transaction'
