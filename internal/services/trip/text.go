package trip

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	maxSlugLength   = 80
	minHandleLength = 3
	maxHandleLength = 30
)

var (
	slugRe   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	handleRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
)

// Slugify turns a title into a slug: lowercase ASCII letters and digits
// separated by single dashes, at most 80 characters long. Accents are
// removed, e.g. "Säntis via the Lisengrat!" becomes
// "santis-via-the-lisengrat". It returns an empty string if the title has
// no letters or digits.
func Slugify(title string) string {
	var b strings.Builder
	dash := false
	// Decompose accented letters into the letter and the accent, and drop
	// the accents.
	for _, r := range norm.NFKD.String(strings.ToLower(title)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		case r == 'ß':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteString("ss")
		default:
			dash = true
		}
	}
	slug := b.String()
	if len(slug) > maxSlugLength {
		slug = slug[:maxSlugLength]
		// Cut at the last dash, rather than in the middle of a word.
		if i := strings.LastIndexByte(slug, '-'); i > maxSlugLength/2 {
			slug = slug[:i]
		}
		slug = strings.TrimRight(slug, "-")
	}
	return slug
}

// validateSlug checks that a slug is lowercase letters and digits
// separated by single dashes.
func validateSlug(slug string) error {
	if slug == "" {
		return errors.New("the slug is required")
	}
	if len(slug) > maxSlugLength {
		return fmt.Errorf("the slug must be at most %d characters", maxSlugLength)
	}
	if !slugRe.MatchString(slug) {
		return errors.New("the slug must be lowercase letters and digits separated by single dashes, e.g. my-first-ultra")
	}
	return nil
}

// validateHandle checks that a handle is 3 to 30 lowercase letters, digits
// and dashes, starting and ending with a letter or digit.
func validateHandle(handle string) error {
	if len(handle) < minHandleLength || len(handle) > maxHandleLength {
		return fmt.Errorf("the handle must be %d to %d characters", minHandleLength, maxHandleLength)
	}
	if !handleRe.MatchString(handle) {
		return errors.New("the handle must be lowercase letters, digits and dashes, starting and ending with a letter or digit")
	}
	return nil
}

// Excerpt returns the start of a text, with whitespace collapsed, cut at a
// word boundary to at most maxLength characters with an ellipsis.
func Excerpt(text string, maxLength int) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	runes := []rune(collapsed)
	if len(runes) <= maxLength {
		return collapsed
	}
	cut := string(runes[:maxLength-1])
	if i := strings.LastIndexByte(cut, ' '); i > len(cut)/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:-") + "…"
}
