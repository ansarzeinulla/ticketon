#!/usr/bin/env bash
# BF-16 Cover users, tokens and mail
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-15 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-16 S1 week-02-ansar-BF-16 origin/week-02-ansar-BF-15 'BF-16 Cover users, tokens and mail'
