#!/usr/bin/env bash
# BF-78 Cut Cyrillic text by characters on the ticket
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-B6 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-78 S2 week-08-alibi-BF-78 origin/week-08-ansar-BF-B6 'BF-78 Cut Cyrillic text by characters on the ticket'
