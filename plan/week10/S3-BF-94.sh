#!/usr/bin/env bash
# BF-94 Update the web documentation
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-94 S3 week-10-abylay-BF-94 origin/week-10 'BF-94 Update the web documentation'
