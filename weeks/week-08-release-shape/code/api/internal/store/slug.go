package store

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// maxSlugLength keeps generated slugs readable in a URL.
const maxSlugLength = 80

// Slugify turns an event title into a URL-safe slug.
//
// Cyrillic titles are common in Kazakhstan, so letters that have no ASCII form
// are transliterated rather than dropped; a title that transliterates to
// nothing yields an empty string and the caller falls back to a default.
func Slugify(title string) string {
	lowered := strings.ToLower(strings.TrimSpace(title))
	transliterated := transliterate(lowered)

	// Strip combining marks left over from decomposition (e.g. é -> e).
	decomposed, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC),
		transliterated)
	if err != nil {
		decomposed = transliterated
	}

	var b strings.Builder
	lastWasHyphen := true // leading hyphens are suppressed
	for _, r := range decomposed {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastWasHyphen = false
		default:
			if !lastWasHyphen {
				b.WriteRune('-')
				lastWasHyphen = true
			}
		}
	}

	slug := strings.Trim(b.String(), "-")
	if len(slug) > maxSlugLength {
		slug = strings.Trim(slug[:maxSlugLength], "-")
	}
	return slug
}

// cyrillic maps Kazakh and Russian letters to their ASCII transliteration.
var cyrillic = map[rune]string{
	'а': "a", 'ә': "a", 'б': "b", 'в': "v", 'г': "g", 'ғ': "g", 'д': "d",
	'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i", 'й': "i", 'к': "k",
	'қ': "q", 'л': "l", 'м': "m", 'н': "n", 'ң': "ng", 'о': "o", 'ө': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ұ': "u", 'ү': "u",
	'ф': "f", 'х': "h", 'һ': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'і': "i", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func transliterate(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if replacement, ok := cyrillic[r]; ok {
			b.WriteString(replacement)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
