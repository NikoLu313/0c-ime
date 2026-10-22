package session

import (
	"testing"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

func TestHsuTypesSyllables(t *testing.T) {
	for keys, want := range map[string]string{
		"xhf": "我", "cekf": "想", "v ": "吃", "cj": "是", "af": "此", "lj": "二",
		"j ": "知", "vuj": "去", "zj": "字", "dgs": "的", "fl jelf": "風景",
		"bm ": "班", "n ": "嗯", "kd": "昂", "mys": "嘛", "xhfcekfv ": "我想吃",
	} {
		s := etenSession(t)
		s.SetLayout(layout.Hsu)
		if snap := typeKeys(s, keys); !snap.Handled || snap.Text() != want {
			t.Errorf("Hsu %q: got %q, want %q", keys, snap.Text(), want)
		}
	}
}

func hsuSharedSession() *Session {
	lex, sys := lexicon.NewMem(), lm.NewMem()
	for _, row := range []struct{ word, reading string }{
		{"詞", "ㄘˊ"}, {"欸", "ㄟˊ"}, {"意", "ㄧˋ"}, {"誒", "ㄝˋ"}, {"兒", "ㄦ"}, {"鞥", "ㄥ"}, {"我", "ㄨㄛˇ"},
	} {
		lex.Add(lexicon.Entry{Surface: row.word, Reading: zhuyin.ParseReading(row.reading), PRW: 1})
		sys.SetUni(row.word, .001)
	}
	s := New(personal.NewModel(lex, sys))
	s.SetLayout(layout.Hsu)
	return s
}

func TestHsuSharedCandidatesAndLearning(t *testing.T) {
	for _, row := range []struct{ keys, primary, alternate, reading string }{
		{"ad", "詞", "欸", "ㄟˊ"}, {"ej", "意", "誒", "ㄝˋ"}, {"l ", "兒", "鞥", "ㄥ"},
	} {
		t.Run(row.keys, func(t *testing.T) {
			s := hsuSharedSession()
			typeKeys(s, row.keys)
			snap := key(s, proto.KeyDown)
			if snap.Candidates == nil {
				t.Fatal("missing shared-key candidates")
			}
			primary, alt := false, -1
			for _, item := range snap.Candidates.Items {
				primary = primary || item.Text == row.primary
				if item.Text == row.alternate {
					alt = item.ID
					if item.Note != "共鍵 "+row.reading {
						t.Fatalf("shared reading labeled as fuzzy: %+v", item)
					}
				}
			}
			if !primary || alt < 0 {
				t.Fatalf("both shared readings must be selectable: %+v", snap.Candidates.Items)
			}
			snap = s.Handle(proto.Request{Type: proto.Select, Index: alt})
			if snap.Text() != row.alternate || s.autoFuzzed() || s.fuzzyTouches(0, 1) {
				t.Fatalf("shared selection is exact: %+v", snap)
			}
			if got := zhuyin.Key(s.learningSyllables()); got != row.reading {
				t.Fatalf("learning reading %q, want %q", got, row.reading)
			}
			if snap = typeKeys(s, "xhf"); snap.Text() != row.alternate+"我" {
				t.Fatalf("shared selection lost when continuing: %q", snap.Text())
			}
			if snap = key(s, proto.KeyEnter); snap.Commit != row.alternate+"我" || s.Composing() {
				t.Fatalf("commit: %+v", snap)
			}
		})
	}
}

func TestHsuPickPendingSharedReading(t *testing.T) {
	for _, row := range []struct{ key, reading, tone, want string }{
		{"a", "ㄟ", "d", "欸"}, {"e", "ㄝ", "j", "誒"}, {"l", "ㄥ", " ", "鞥"},
	} {
		s := hsuSharedSession()
		key(s, row.key)
		snap := key(s, proto.KeyDown)
		if snap.Candidates == nil {
			t.Fatal("missing pending reading choices")
		}
		id := -1
		for _, item := range snap.Candidates.Items {
			if item.Text == row.reading {
				id = item.ID
			}
		}
		if id < 0 {
			t.Fatalf("missing %q: %+v", row.reading, snap.Candidates.Items)
		}
		s.Handle(proto.Request{Type: proto.Select, Index: id})
		if s.comp.String() != row.reading || len(s.syls) != 0 {
			t.Fatal("reading choice must remain editable phonetics")
		}
		if snap := key(s, row.tone); snap.Text() != row.want {
			t.Fatalf("reading then tone: got %q, want %q", snap.Text(), row.want)
		}
	}
}

func TestHsuAllStandaloneVowelTones(t *testing.T) {
	for _, base := range []zhuyin.Syllable{"ㄚ", "ㄛ", "ㄜ", "ㄝ", "ㄞ", "ㄟ", "ㄠ", "ㄡ", "ㄢ", "ㄣ", "ㄤ", "ㄥ", "ㄦ", "ㄧ", "ㄨ", "ㄩ"} {
		for tone := 1; tone <= 5; tone++ {
			syl := base.WithTone(tone)
			lex, sys := lexicon.NewMem(), lm.NewMem()
			lex.Add(lexicon.Entry{Surface: "測", Reading: []zhuyin.Syllable{syl}, PRW: 1})
			sys.SetUni("測", .01)
			s := New(personal.NewModel(lex, sys))
			s.SetLayout(layout.Hsu)
			snap := typeKeys(s, layout.Hsu.Keys(syl))
			if snap.Text() != "測" {
				t.Fatalf("standalone %q keys %q -> %q", syl, layout.Hsu.Keys(syl), snap.Text())
			}
		}
	}
}

func TestHsuEditingAndSwitchBack(t *testing.T) {
	s := etenSession(t)
	s.SetLayout(layout.Hsu)
	s.MixedInput = true
	if s.mixedInput() {
		t.Fatal("Hsu mixed input must remain disabled")
	}
	typeKeys(s, "ce")
	key(s, proto.KeyBackspace)
	if snap := typeKeys(s, "j"); snap.Text() != "是" {
		t.Fatalf("backspace then tone: %q", snap.Text())
	}
	if snap := key(s, "'"); snap.Text() != "是、" {
		t.Fatalf("punctuation: %q", snap.Text())
	}
	key(s, proto.KeyEscape)
	s.SetLayout(layout.Standard)
	s.MixedInput = false
	if snap := typeKeys(s, "su3"); snap.Text() != "ㄋㄧˇ" {
		// 測試詞庫沒有「你」，保底仍應顯示標準排列的注音。
		t.Fatalf("standard after Hsu: %q", snap.Text())
	}
	key(s, proto.KeyEscape)
	s.SetLayout(layout.ETen)
	if snap := typeKeys(s, "xo3"); snap.Text() != "我" {
		t.Fatalf("ETen after Hsu: %q", snap.Text())
	}
}

func TestHsuUnusedIdleKeysPassThrough(t *testing.T) {
	for _, k := range []string{"q", "1", "2", "3", "4", "5", "6", "7", "8", "9", "0"} {
		s := etenSession(t)
		s.SetLayout(layout.Hsu)
		if snap := key(s, k); snap.Handled || s.Composing() {
			t.Fatalf("Hsu unused idle key %q must pass through: %+v", k, snap)
		}
	}
}
