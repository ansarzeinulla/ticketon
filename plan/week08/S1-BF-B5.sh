#!/usr/bin/env bash
# BF-B5 Stop sharing notification state on the server
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-77 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B5 S1 week-08-ansar-BF-B5 origin/week-08-ansar-BF-77 'BF-B5 Stop sharing notification state on the server'
