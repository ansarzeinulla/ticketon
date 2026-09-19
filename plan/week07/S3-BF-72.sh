#!/usr/bin/env bash
# BF-72 Localise the attendee pages
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-71 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-72 S3 week-07-abylay-BF-72 origin/week-07-abylay-BF-71 'BF-72 Localise the attendee pages'
