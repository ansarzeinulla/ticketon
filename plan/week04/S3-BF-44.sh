#!/usr/bin/env bash
# BF-44 Add the seat map and seated checkout
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-43 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-44 S3 week-04-abylay-BF-44 origin/week-04-abylay-BF-43 'BF-44 Add the seat map and seated checkout'
