#!/usr/bin/env bash
# BF-B6 Make the seed re-runnable after a purchase
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B5 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B6 S1 week-08-ansar-BF-B6 origin/week-08-ansar-BF-B5 'BF-B6 Make the seed re-runnable after a purchase'
