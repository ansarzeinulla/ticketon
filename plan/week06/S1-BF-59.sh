#!/usr/bin/env bash
# BF-59 Add the report queue and settings
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B1 (Olzhas Nurseit): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-59 S1 week-06-ansar-BF-59 origin/week-06-olzhas-BF-B1 'BF-59 Add the report queue and settings'
