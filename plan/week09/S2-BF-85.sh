#!/usr/bin/env bash
# BF-85 Refuse to publish an event that has ended
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-85 S2 week-09-alibi-BF-85 origin/week-09 'BF-85 Refuse to publish an event that has ended'
