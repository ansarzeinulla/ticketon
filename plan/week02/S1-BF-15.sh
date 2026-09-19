#!/usr/bin/env bash
# BF-15 Reject duplicate emails with 409
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-14 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-15 S1 week-02-ansar-BF-15 origin/week-02-ansar-BF-14 'BF-15 Reject duplicate emails with 409'
