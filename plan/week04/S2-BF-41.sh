#!/usr/bin/env bash
# BF-41 Gate paid sales behind activation
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-40 (Alibi Takhtanov): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-41 S2 week-04-alibi-BF-41 origin/week-04-alibi-BF-40 'BF-41 Gate paid sales behind activation'
