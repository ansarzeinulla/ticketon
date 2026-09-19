#!/usr/bin/env bash
# BF-83 Point the scanner at the queue
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B10 (Olzhas Nurseit): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-83 S5 week-08-olzhas-BF-83 origin/week-08-olzhas-BF-B10 'BF-83 Point the scanner at the queue'
