#!/usr/bin/env bash
# BF-45 Link the printable PDF
# Запускает Abylay Otaubay (S3) сам, с этого компьютера, из любой папки.
# Строится поверх BF-44 (Abylay Otaubay): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-45 S3 week-04-abylay-BF-45 origin/week-04-abylay-BF-44 'BF-45 Link the printable PDF'
