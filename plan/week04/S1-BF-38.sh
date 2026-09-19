#!/usr/bin/env bash
# BF-38 Enforce role permissions
# Запускает Ansar Zeinulla (S1) сам, с этого компьютера, из любой папки.
# Строится поверх BF-37 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-38 S1 week-04-ansar-BF-38 origin/week-04-ansar-BF-37 'BF-38 Enforce role permissions'
