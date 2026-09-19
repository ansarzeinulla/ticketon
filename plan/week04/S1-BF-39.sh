#!/usr/bin/env bash
# BF-39 Send the order confirmation
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-41 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-39 S1 week-04-ansar-BF-39 origin/week-04-alibi-BF-41 'BF-39 Send the order confirmation'
