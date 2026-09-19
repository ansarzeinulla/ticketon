#!/usr/bin/env bash
# BF-90 Correct the specifications after the run
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-87 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-90 S5 week-09-olzhas-BF-90 origin/week-09-abylay-BF-87 'BF-90 Correct the specifications after the run'
