#!/usr/bin/env bash
# BF-25 Derive a slug from the title
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-23 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-25 S2 week-03-alibi-BF-25 origin/week-03-ansar-BF-23 'BF-25 Derive a slug from the title'
