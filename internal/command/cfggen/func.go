package cfggen

import (
	"fmt"
	"strings"
	"text/template"
)

var UseFuncMap = template.FuncMap{}

func init() {
	UseFuncMap["CamelCase"] = CamelCase
	UseFuncMap["Comment"] = func(f *optionField) (out string) {
		out = f.Doc()
		if strings.Contains(out, "\n") {
			list := strings.Split(out, "\n")
			out = ""
			for _, v := range list {
				out += "// " + v + "\n"
			}
			return
		}
		if len(out) > 0 {
			return "// " + out + "\n"
		}
		return
	}

	UseFuncMap["Doc"] = func(doc string, comments []string) string {
		if len(doc) > 0 {
			return strings.TrimSpace(doc)
		}
		if len(comments) > 0 {
			return strings.TrimSpace(comments[0])
		}
		return ""
	}

	UseFuncMap["OneRow"] = func(in string) string {
		if strings.Contains(in, "\n") {
			return strings.Replace(in, "\n", " ", -1)
		}
		return in
	}

	UseFuncMap["ToLower"] = func(in string) string {
		if config.Lowercase {
			return strings.ToLower(in)
		}
		return in
	}
	//UseFuncMap["RegName"] = GetRegName
	UseFuncMap["Tag"] = func(f string, v ...string) string {
		for k := range v {
			v[k] = strings.ToLower(v[k])
		}
		return fmt.Sprintf("`%s:\"%s\"`", strings.ToLower(f), strings.Join(v, ","))
	}
}

func CamelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
