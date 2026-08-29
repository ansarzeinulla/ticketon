package ticketpdf

import (
	"strings"
	"unicode"
)

// cyrillic maps Kazakh and Russian letters to an ASCII transliteration.
//
// The PDF uses the built-in Helvetica font, whose cp1252 encoding has no
// Cyrillic. Rather than emit mojibake, text outside cp1252 is transliterated so
// the ticket stays readable. Embedding a Unicode font is the proper fix and is
// noted in the README; until then this degrades gracefully instead of silently
// producing an unreadable ticket.
var cyrillic = map[rune]string{
	'а': "a", 'ә': "a", 'б': "b", 'в': "v", 'г': "g", 'ғ': "g", 'д': "d",
	'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i", 'й': "i", 'к': "k",
	'қ': "q", 'л': "l", 'м': "m", 'н': "n", 'ң': "ng", 'о': "o", 'ө': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ұ': "u", 'ү': "u",
	'ф': "f", 'х': "h", 'һ': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'і': "i", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// pdfSafe returns text the core PDF font can actually render.
//
// Characters that exist in cp1252 (which covers Latin-1 accents) are kept as
// they are; anything else is transliterated, and whatever still cannot be
// mapped is dropped rather than drawn as a wrong glyph.
func pdfSafe(text string) string {
	var b strings.Builder
	b.Grow(len(text))

	for _, r := range text {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		case inCP1252(r):
			b.WriteRune(r)
		default:
			lower := unicode.ToLower(r)
			replacement, ok := cyrillic[lower]
			if !ok {
				continue
			}
			if r != lower && replacement != "" {
				// Preserve the original capitalisation.
				replacement = strings.ToUpper(replacement[:1]) + replacement[1:]
			}
			b.WriteString(replacement)
		}
	}

	return b.String()
}

// inCP1252 reports whether the rune has a cp1252 code point.
func inCP1252(r rune) bool {
	if r >= 0xA0 && r <= 0xFF {
		return true
	}
	// The scattered printable characters cp1252 puts in 0x80-0x9F.
	switch r {
	case '€', '‚', 'ƒ', '„', '…', '†', '‡',
		'ˆ', '‰', 'Š', '‹', 'Œ', 'Ž', '‘',
		'’', '“', '”', '•', '–', '—', '˜',
		'™', 'š', '›', 'œ', 'ž', 'Ÿ':
		return true
	}
	return false
}

// truncate shortens text to fit a fixed-width field, with an ellipsis.
func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return strings.TrimRight(string(runes[:max-1]), " ") + "..."
}
