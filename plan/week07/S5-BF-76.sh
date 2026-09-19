#!/usr/bin/env bash
# BF-76 Add the Phase 7 specification
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B4 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-76 S5 week-07-olzhas-BF-76 origin/week-07-alinur-BF-B4 'BF-76 Add the Phase 7 specification'
