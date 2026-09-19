#!/usr/bin/env bash
# BF-B1 Keep the check-in record when a ticket is voided
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-62 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-B1 S5 week-06-olzhas-BF-B1 origin/week-06-alibi-BF-62 'BF-B1 Keep the check-in record when a ticket is voided'
