#!/usr/bin/env bash
# BF-73 Add the language negotiation dependency
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-72 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-73 S3 week-07-abylay-BF-73 origin/week-07-abylay-BF-72 'BF-73 Add the language negotiation dependency'
