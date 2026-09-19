#!/usr/bin/env bash
# BF-67 Add the Phase 6 specification
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-65 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-67 S5 week-06-olzhas-BF-67 origin/week-06-alinur-BF-65 'BF-67 Add the Phase 6 specification'
