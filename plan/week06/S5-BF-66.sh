#!/usr/bin/env bash
# BF-66 Refuse a refunded ticket at the gate
# Запускает Olzhas Nurseit (S5) сам, с этого компьютера, из любой папки.
# Строится поверх BF-61 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-66 S5 week-06-olzhas-BF-66 origin/week-06-alibi-BF-61 'BF-66 Refuse a refunded ticket at the gate'
