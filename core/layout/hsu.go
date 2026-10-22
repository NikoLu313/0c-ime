package layout

import "github.com/0c-biggo/0c-ime/core/zhuyin"

// Hsu 是許氏排列。共鍵規則對照 libchewing 的許氏行為：
// https://github.com/chewing/libchewing/blob/3c4a93aa03d574c7f011ff84e8a2437c2f79b2cf/src/editor/zhuyin_layout/hsu.rs
// Symbol 只回報閒置鍵義；實際逐鍵輸入必須使用 ResolveSymbol。
// 獨立 ㄟ／ㄝ／ㄥ 與 ㄘ／ㄧ／ㄦ 共鍵，由 Alternatives 展開讀音供選字。
var Hsu Layout = newHsu()

type hsuLayout struct{ *table }

func newHsu() *hsuLayout {
	punct := make(map[rune]string, len(standardPunct)+5)
	for key, text := range standardPunct {
		punct[key] = text
	}
	for key, text := range map[rune]string{',': "，", '.': "。", '/': "／", ';': "；", '-': "－"} {
		punct[key] = text
	}
	t := withPunct(punct, NameHsu,
		"bㄅ", "pㄆ", "mㄇ", "fㄈ", "dㄉ", "tㄊ", "nㄋ", "lㄌ", "gㄍ", "kㄎ", "hㄏ",
		"jㄐ", "vㄑ", "cㄒ", "jㄓ", "vㄔ", "cㄕ", "rㄖ", "zㄗ", "aㄘ", "sㄙ",
		"eㄧ", "xㄨ", "uㄩ",
		"yㄚ", "hㄛ", "gㄜ", "eㄝ", "iㄞ", "aㄟ", "wㄠ", "oㄡ", "mㄢ", "nㄣ", "kㄤ", "lㄥ", "lㄦ",
		" ˉ", "dˊ", "fˇ", "jˋ", "s˙",
	)
	// 共鍵的閒置義：聲母優先；e 是介音 ㄧ。
	for _, p := range []string{"mㄇ", "fㄈ", "dㄉ", "nㄋ", "lㄌ", "gㄍ", "kㄎ", "hㄏ", "jㄓ", "aㄘ", "sㄙ", "eㄧ"} {
		r := []rune(p)
		t.sym[r[0]] = r[1]
	}
	return &hsuLayout{t}
}

// Alternatives 回傳與 syl 使用相同實體按鍵的其他完整音節（不含 syl）。
// 共鍵是精確輸入的另一個解讀，不屬於模糊音；聲調保留不變。
func Alternatives(l Layout, syl zhuyin.Syllable) []zhuyin.Syllable {
	if l != Hsu || syl == "" {
		return nil
	}
	tone := syl.ToneOf()
	base := string(syl.WithTone(1))
	var alt string
	switch base {
	case "ㄘ":
		alt = "ㄟ"
	case "ㄟ":
		alt = "ㄘ"
	case "ㄧ":
		alt = "ㄝ"
	case "ㄝ":
		alt = "ㄧ"
	case "ㄦ":
		alt = "ㄥ"
	case "ㄥ":
		alt = "ㄦ"
	default:
		return nil
	}
	return []zhuyin.Syllable{zhuyin.Syllable(alt).WithTone(tone)}
}

func (h *hsuLayout) resolve(key rune, c *zhuyin.Composer) (rune, bool) {
	sym, ok := h.Symbol(key)
	if !ok {
		return 0, false
	}
	initial, medial, final := c.Slot(zhuyin.Initial), c.Slot(zhuyin.Medial), c.Slot(zhuyin.Final)
	if !c.Empty() {
		var tone rune
		switch key {
		case ' ':
			tone = zhuyin.Tone1
		case 'd':
			tone = 'ˊ'
		case 'f':
			tone = 'ˇ'
		case 'j':
			tone = 'ˋ'
		case 's':
			tone = '˙'
		}
		if tone != 0 {
			// 單獨的共鍵聲母在收音時改讀韻母。
			if medial == 0 && final == 0 {
				if r := map[rune]rune{'ㄐ': 'ㄓ', 'ㄑ': 'ㄔ', 'ㄒ': 'ㄕ'}[initial]; r != 0 {
					c.Add(r)
				}
				if r := map[rune]rune{'ㄏ': 'ㄛ', 'ㄍ': 'ㄜ', 'ㄇ': 'ㄢ', 'ㄋ': 'ㄣ', 'ㄎ': 'ㄤ', 'ㄌ': 'ㄦ'}[initial]; r != 0 {
					c.Reset()
					c.Add(r)
				}
			}
			if initial == 'ㄍ' && (medial == 'ㄧ' || medial == 'ㄩ') {
				c.Add('ㄐ')
			}
			return tone, true
		}
	}
	if initial != 0 || medial != 0 {
		if r := map[rune]rune{'a': 'ㄟ', 'g': 'ㄜ', 'h': 'ㄛ', 'k': 'ㄤ', 'l': 'ㄥ', 'm': 'ㄢ', 'n': 'ㄣ'}[key]; r != 0 {
			sym = r
		}
	}
	if key == 'e' && medial != 0 {
		sym = 'ㄝ'
	}
	// ㄓㄔㄕ／ㄐㄑㄒ共鍵，以介音決定聲母。
	if sym == 'ㄧ' || sym == 'ㄩ' {
		if r := map[rune]rune{'ㄓ': 'ㄐ', 'ㄔ': 'ㄑ', 'ㄕ': 'ㄒ', 'ㄍ': 'ㄐ'}[initial]; r != 0 {
			c.Add(r)
		}
	} else if sym == 'ㄨ' || (zhuyin.ClassOf(sym) == zhuyin.Final && medial == 0) {
		if r := map[rune]rune{'ㄐ': 'ㄓ', 'ㄑ': 'ㄔ', 'ㄒ': 'ㄕ'}[initial]; r != 0 {
			c.Add(r)
		}
	}
	return sym, true
}
