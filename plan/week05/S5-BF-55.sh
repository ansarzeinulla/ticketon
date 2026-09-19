#!/usr/bin/env bash
# BF-55 Prepare the gate backend and a fake scan payload
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-50 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-55 S5 week-05-olzhas-BF-55 origin/week-05-ansar-BF-50 'BF-55 Prepare the gate backend and a fake scan payload'
