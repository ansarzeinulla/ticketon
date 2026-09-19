#!/usr/bin/env bash
# BF-62 Expose the rows the admin search needs
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-66 (Olzhas Nurseit): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-62 S2 week-06-alibi-BF-62 origin/week-06-olzhas-BF-66 'BF-62 Expose the rows the admin search needs'
