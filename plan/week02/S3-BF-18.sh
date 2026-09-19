#!/usr/bin/env bash
# BF-18 Add the API client
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-18 S3 week-02-abylay-BF-18 origin/week-02 'BF-18 Add the API client'
