#!/usr/bin/env bash
# BF-56 Scan and admit a ticket
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-55 (Olzhas Nurseit): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-56 S5 week-05-olzhas-BF-56 origin/week-05-olzhas-BF-55 'BF-56 Scan and admit a ticket'
