#!/usr/bin/env bash
# BF-42 Embed a Unicode font for Cyrillic
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-39 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-42 S2 week-04-alibi-BF-42 origin/week-04-ansar-BF-39 'BF-42 Embed a Unicode font for Cyrillic'
