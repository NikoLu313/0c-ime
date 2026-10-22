package layout

import (
	"slices"
	"testing"

	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func TestHsuContext(t *testing.T) {
	for keys, want := range map[string]zhuyin.Syllable{
		"xhf": "ㄨㄛˇ", "cekf": "ㄒㄧㄤˇ", "v ": "ㄔ", "cj": "ㄕˋ",
		"af": "ㄘˇ", "lj": "ㄦˋ", "juj": "ㄐㄩˋ", "vuj": "ㄑㄩˋ",
		"j ": "ㄓ", "vif": "ㄔㄞˇ", "ce ": "ㄒㄧ", "jxf": "ㄓㄨˇ",
		"dgs": "ㄉㄜ˙", "fl ": "ㄈㄥ", "jelf": "ㄐㄧㄥˇ",
		"bm ": "ㄅㄢ", "n ": "ㄣ", "kd": "ㄤˊ", "mys": "ㄇㄚ˙",
		"h ": "ㄛ", "g ": "ㄜ", "m ": "ㄢ", "eej": "ㄧㄝˋ",
		"ge ": "ㄐㄧ", "guj": "ㄐㄩˋ", "baj": "ㄅㄟˋ",
	} {
		t.Run(keys, func(t *testing.T) {
			var c zhuyin.Composer
			var got zhuyin.Syllable
			for _, key := range keys {
				sym, ok := ResolveSymbol(Hsu, key, &c)
				if !ok {
					t.Fatalf("unmapped key %q", key)
				}
				if done, ok := c.Add(sym); ok {
					got = done
				}
			}
			if got != want || !c.Empty() {
				t.Fatalf("%q: got %q (pending %q), want %q", keys, got, c.String(), want)
			}
			// Keys 必須能走相同的共鍵處理路徑。
			for _, key := range Hsu.Keys(want) {
				sym, _ := ResolveSymbol(Hsu, key, &c)
				if done, ok := c.Add(sym); ok {
					got = done
				}
			}
			if got != want || !c.Empty() {
				t.Fatalf("Keys(%q)=%q did not round trip: %q", want, Hsu.Keys(want), got)
			}
		})
	}
}

func TestHsuEveryLegalSyllableReachable(t *testing.T) {
	// 窮舉國語音節表，確認未因共鍵規則讓合法讀音無法輸入。
	for _, initial := range append([]rune{0}, []rune("ㄅㄆㄇㄈㄉㄊㄋㄌㄍㄎㄏㄐㄑㄒㄓㄔㄕㄖㄗㄘㄙ")...) {
		for _, medial := range []rune{0, 'ㄧ', 'ㄨ', 'ㄩ'} {
			for _, final := range append([]rune{0}, []rune("ㄚㄛㄜㄝㄞㄟㄠㄡㄢㄣㄤㄥㄦ")...) {
				var symbols []rune
				for _, sym := range []rune{initial, medial, final} {
					if sym != 0 {
						symbols = append(symbols, sym)
					}
				}
				base := zhuyin.Syllable(symbols)
				if !base.Legal() {
					continue
				}
				for tone := 1; tone <= 5; tone++ {
					want := base.WithTone(tone)
					var c zhuyin.Composer
					var done zhuyin.Syllable
					for _, key := range Hsu.Keys(want) {
						sym, ok := ResolveSymbol(Hsu, key, &c)
						if !ok {
							t.Fatalf("unmapped key %q for %q", key, want)
						}
						if syl, ok := c.Add(sym); ok {
							done = syl
						}
					}
					if !c.Empty() || done != want && !slices.Contains(Alternatives(Hsu, done), want) {
						t.Fatalf("%q is unreachable: keys %q -> %q, pending %q", want, Hsu.Keys(want), done, c.String())
					}
				}
			}
		}
	}
}

func TestHsuIdleAndEditing(t *testing.T) {
	for key, want := range map[rune]rune{'d': 'ㄉ', 'f': 'ㄈ', 'j': 'ㄓ', 's': 'ㄙ'} {
		var c zhuyin.Composer
		got, ok := ResolveSymbol(Hsu, key, &c)
		if !ok || got != want || !c.Empty() {
			t.Fatalf("idle %q = %q, %v", key, got, ok)
		}
	}
	var a, b zhuyin.Composer
	for _, key := range "ce" {
		sym, _ := ResolveSymbol(Hsu, key, &a)
		a.Add(sym)
	}
	a.Backspace()
	sym, _ := ResolveSymbol(Hsu, 'x', &a)
	a.Add(sym)
	if a.String() != "ㄕㄨ" || !b.Empty() {
		t.Fatalf("editing/state isolation: %q, %q", a.String(), b.String())
	}
	before := a.String()
	if _, ok := ResolveSymbol(Hsu, '=', &a); ok || a.String() != before {
		t.Fatal("unknown key changed composer")
	}
	if p, ok := Hsu.Punct('\''); !ok || p != "、" {
		t.Fatal("Hsu punctuation")
	}
}

func TestHsuSharedReadings(t *testing.T) {
	for _, pair := range [][2]zhuyin.Syllable{{"ㄘ", "ㄟ"}, {"ㄧ", "ㄝ"}, {"ㄦ", "ㄥ"}} {
		for tone := 1; tone <= 5; tone++ {
			a, b := pair[0].WithTone(tone), pair[1].WithTone(tone)
			if Hsu.Keys(a) != Hsu.Keys(b) || Hsu.Keys(a) == "" {
				t.Fatalf("shared keys for %q/%q: %q/%q", a, b, Hsu.Keys(a), Hsu.Keys(b))
			}
			if got := Alternatives(Hsu, a); len(got) != 1 || got[0] != b {
				t.Fatalf("Alternatives(%q) = %v, want %q", a, got, b)
			}
			if got := Alternatives(Hsu, b); len(got) != 1 || got[0] != a {
				t.Fatalf("Alternatives(%q) = %v, want %q", b, got, a)
			}
		}
	}
	for _, l := range []Layout{Standard, ETen} {
		if got := Alternatives(l, "ㄧ"); len(got) != 0 {
			t.Fatalf("%s must stay exact: %v", l.Name(), got)
		}
	}
	if got := Alternatives(Hsu, "ㄒㄧㄤˇ"); len(got) != 0 {
		t.Fatalf("unambiguous syllable: %v", got)
	}
	for key, want := range map[rune]string{',': "，", '.': "。", '/': "／", ';': "；", '-': "－"} {
		if got, ok := Hsu.Punct(key); !ok || got != want {
			t.Fatalf("Hsu punctuation %q: %q, %v", key, got, ok)
		}
	}
}
