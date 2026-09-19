#!/usr/bin/env bash
# BF-51 Reserve stock with 15-minute holds
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-51 S2 week-05-alibi-BF-51 origin/week-05 'BF-51 Reserve stock with 15-minute holds'
