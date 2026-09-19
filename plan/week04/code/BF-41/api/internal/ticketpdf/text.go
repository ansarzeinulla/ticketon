package ticketpdf

import (
	"strings"
	"unicode"
)

// cyrillic maps Kazakh and Russian letters to an ASCII approximation.
//
// The core PDF fonts are cp1252 and carry no Cyrillic, so a name written in
// Kazakh would otherwise print as a row of question marks. Transliterating is
// readable, if not the attendee's actual name.
var cyrillic = map[rune]string{
	'а': "a", 'ә': "a", 'б': "b", 'в': "v", 'г': "g", 'ғ': "g", 'д': "d",
	'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i", 'й': "i", 'к': "k",
	'қ': "q", 'л': "l", 'м': "m", 'н': "n", 'ң': "ng", 'о': "o", 'ө': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ұ': "u", 'ү': "u",
	'ф': "f", 'х': "h", 'һ': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'і': "i", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// transliterate rewrites Cyrillic into the Latin letters the PDF font has.
// A capital letter keeps its case on the first letter only, so "Жалпы" reads
// "Zhalpy" rather than "ZHalpy".
func transliterate(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		lower := unicode.ToLower(r)
		replacement, ok := cyrillic[lower]
		if !ok {
			b.WriteRune(r)
			continue
		}
		if lower != r && replacement != "" {
			replacement = strings.ToUpper(replacement[:1]) + replacement[1:]
		}
		b.WriteString(replacement)
	}
	return b.String()
}

// truncate shortens text to at most max bytes, ending with an ellipsis.
func truncate(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return strings.TrimRight(text[:max-1], " ") + "..."
}
