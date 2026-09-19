#!/usr/bin/env bash
# BF-61 Refund an order atomically
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-61 S2 week-06-alibi-BF-61 origin/week-06 'BF-61 Refund an order atomically'
