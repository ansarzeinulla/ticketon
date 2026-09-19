#!/usr/bin/env bash
# BF-93 Update the API documentation
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-93 S2 week-10-alibi-BF-93 origin/week-10 'BF-93 Update the API documentation'
