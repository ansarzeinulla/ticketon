#!/usr/bin/env bash
# BF-64 Point the client at the proxy
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-60 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-64 S3 week-06-abylay-BF-64 origin/week-06-ansar-BF-60 'BF-64 Point the client at the proxy'
