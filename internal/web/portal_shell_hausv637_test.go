package web

import (
	"os"
	"strings"
	"testing"
)

// cssRule is one selector block of a stylesheet together with the at-rule
// preludes it sits in, outermost first. Declarations are kept verbatim.
type cssRule struct {
	context      []string
	selectors    []string
	declarations string
}

// parseCSSRules walks minified or formatted CSS and lists every selector block
// with its enclosing at-rules. It understands comments, nested at-rules and
// prelude-only at-rules (@import …;), which is all the shell stylesheet uses.
func parseCSSRules(css string) []cssRule {
	var rules []cssRule
	var context []string
	buf := strings.Builder{}
	for i := 0; i < len(css); {
		switch {
		case strings.HasPrefix(css[i:], "/*"):
			end := strings.Index(css[i+2:], "*/")
			if end < 0 {
				return rules
			}
			i += end + 4
		case css[i] == '@':
			end := strings.IndexAny(css[i:], "{;")
			if end < 0 {
				return rules
			}
			prelude := strings.Join(strings.Fields(css[i:i+end]), " ")
			if css[i+end] == '{' {
				context = append(context, prelude)
			}
			buf.Reset()
			i += end + 1
		case css[i] == '{':
			selectors := strings.Split(buf.String(), ",")
			for k := range selectors {
				selectors[k] = strings.TrimSpace(selectors[k])
			}
			buf.Reset()
			depth, j := 1, i+1
			for j < len(css) && depth > 0 {
				switch css[j] {
				case '{':
					depth++
				case '}':
					depth--
				}
				j++
			}
			rules = append(rules, cssRule{
				context:      append([]string(nil), context...),
				selectors:    selectors,
				declarations: strings.Join(strings.Fields(css[i+1:j-1]), ""),
			})
			i = j
		case css[i] == '}':
			if len(context) > 0 {
				context = context[:len(context)-1]
			}
			buf.Reset()
			i++
		default:
			buf.WriteByte(css[i])
			i++
		}
	}
	return rules
}

func (r cssRule) matches(selector string) bool {
	for _, s := range r.selectors {
		if s == selector {
			return true
		}
	}
	return false
}

// TestPhoneChromeIsHiddenOnlyAboveTheShellBreakpointHAUSV637 guards the
// cascade that blanked the Hausüberblick on phones. portal-shell.css is linked
// after every inline page block, so a rule there beats a page rule of equal
// specificity. An unconditional `.mobile-content{display:none}` in the shell
// therefore outranks the page's `@media(max-width:760px){.mobile-content
// {display:block}}` — the phone layout never appears, at any width. The shell
// may hide the phone chrome only where the desktop shell is in charge, i.e.
// behind a min-width query, and must never hide it inside a max-width block.
func TestPhoneChromeIsHiddenOnlyAboveTheShellBreakpointHAUSV637(t *testing.T) {
	css, err := os.ReadFile("assets/portal-shell.css")
	if err != nil {
		t.Fatalf("read portal-shell.css: %v", err)
	}
	rules := parseCSSRules(string(css))
	if len(rules) < 50 {
		t.Fatalf("parsed only %d rules from portal-shell.css; the parser lost the sheet", len(rules))
	}

	for _, selector := range []string{".mobile-head", ".mobile-content", ".thumb-zone"} {
		hidden := 0
		for _, rule := range rules {
			if !rule.matches(selector) || !strings.Contains(rule.declarations, "display:none") {
				continue
			}
			hidden++
			context := strings.Join(rule.context, " ")
			if !strings.Contains(context, "min-width") {
				t.Errorf("portal-shell.css hides %s without a min-width query (context %q); the phone layout below the breakpoint loses the cascade", selector, context)
			}
			if strings.Contains(context, "max-width") {
				t.Errorf("portal-shell.css hides %s inside a max-width query (context %q)", selector, context)
			}
		}
		if hidden == 0 {
			t.Errorf("portal-shell.css no longer hides %s on desktop; the phone chrome would render beside the sidebar", selector)
		}
	}
}

func TestParseCSSRulesTracksNestedContexts(t *testing.T) {
	rules := parseCSSRules(`/* c */.a,.b{display:none}@media(max-width:760px){.b{display:block}@supports(x:y){.c{color:red}}}.d{x:1}`)
	want := []struct {
		selector, context, declarations string
	}{
		{".a", "", "display:none"},
		{".b", "@media(max-width:760px)", "display:block"},
		{".c", "@media(max-width:760px) @supports(x:y)", "color:red"},
		{".d", "", "x:1"},
	}
	if len(rules) != len(want) {
		t.Fatalf("parsed %d rules, want %d: %+v", len(rules), len(want), rules)
	}
	for i, w := range want {
		got := rules[i]
		if !got.matches(w.selector) || strings.Join(got.context, " ") != w.context || got.declarations != w.declarations {
			t.Errorf("rule %d = %+v, want %+v", i, got, w)
		}
	}
}
