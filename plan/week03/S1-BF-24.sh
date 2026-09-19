#!/usr/bin/env bash
# BF-24 Add order, ticket and attendee tables
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-24 S1 week-03-ansar-BF-24 origin/week-03 'BF-24 Add order, ticket and attendee tables'
