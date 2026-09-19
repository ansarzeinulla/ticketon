#!/usr/bin/env bash
# BF-60 Give every conflict its own error code
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-59 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-60 S1 week-06-ansar-BF-60 origin/week-06-ansar-BF-59 'BF-60 Give every conflict its own error code'
