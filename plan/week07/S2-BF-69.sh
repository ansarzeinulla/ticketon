#!/usr/bin/env bash
# BF-69 Create campaigns and promo codes
# Запускает Alibi Takhtanov (S2) сам, с этого компьютера, из любой папки.
# Строится поверх BF-68 (Ansar Zeinulla): сначала должен быть запущен его скрипт.
exec bash "$(dirname "${BASH_SOURCE[0]}")/_commit.sh" BF-69 S2 week-07-alibi-BF-69 origin/week-07-ansar-BF-68 'BF-69 Create campaigns and promo codes'
