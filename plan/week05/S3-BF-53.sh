#!/usr/bin/env bash
# BF-53 Show the hold countdown at checkout
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-54 (Alinur Burlybayev): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-53 S3 week-05-abylay-BF-53 origin/week-05-alinur-BF-54 'BF-53 Show the hold countdown at checkout'
