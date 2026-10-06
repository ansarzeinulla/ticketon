package store

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"simple", "Almaty Winter Jazz Night", "almaty-winter-jazz-night"},
		{"punctuation", "Rock & Roll: Live!", "rock-roll-live"},
		{"collapses spaces", "  Too   Many   Spaces  ", "too-many-spaces"},
		{"digits kept", "Meetup 2026", "meetup-2026"},
		{"accents folded", "Café Déjà Vu", "cafe-deja-vu"},
		{"russian", "Концерт в Алматы", "kontsert-v-almaty"},
		{"kazakh letters", "Қыс кеші", "qys-keshi"},
		{"already a slug", "already-a-slug", "already-a-slug"},
		{"leading and trailing junk", "---Hello---", "hello"},
		{"emoji only", "🎵🎶", ""},
		{"empty", "", ""},
		{"only punctuation", "!!!???", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Slugify(tt.title); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestSlugifyTruncatesLongTitles(t *testing.T) {
	long := ""
	for i := 0; i < 40; i++ {
		long += "word "
	}

	got := Slugify(long)
	if len(got) > maxSlugLength {
		t.Errorf("Slugify() produced %d characters, want at most %d", len(got), maxSlugLength)
	}
	if got == "" {
		t.Error("Slugify() returned empty for a long but valid title")
	}
	if got[len(got)-1] == '-' {
		t.Errorf("Slugify() = %q, want no trailing hyphen after truncation", got)
	}
}
