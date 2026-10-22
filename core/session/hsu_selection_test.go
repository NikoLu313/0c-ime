package session

import (
	"slices"
	"testing"

	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/lexicon"
	"github.com/0c-biggo/0c-ime/core/lm"
	"github.com/0c-biggo/0c-ime/core/personal"
	"github.com/0c-biggo/0c-ime/core/proto"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

type hsuSelectionWord struct {
	word, reading string
	prob          float64
}

func hsuSelectionSession(words ...hsuSelectionWord) (*Session, *personal.Model) {
	lex, sys := lexicon.NewMem(), lm.NewMem()
	for _, word := range words {
		lex.Add(lexicon.Entry{Surface: word.word, Reading: zhuyin.ParseReading(word.reading), PRW: 1})
		sys.SetUni(word.word, word.prob)
	}
	model := personal.NewModel(lex, sys)
	s := New(model)
	s.SetLayout(layout.Hsu)
	return s, model
}

func hsuChooseText(t *testing.T, s *Session, snap proto.Snapshot, text string) proto.Snapshot {
	t.Helper()
	if snap.Candidates == nil {
		t.Fatalf("missing candidates for %q", text)
	}
	for _, item := range snap.Candidates.Items {
		if item.Text == text {
			return s.Handle(proto.Request{Type: proto.Select, Index: item.ID})
		}
	}
	t.Fatalf("missing %q in %+v", text, snap.Candidates.Items)
	return proto.Snapshot{}
}

// A reading chosen before its tone is an explicit instruction. A much more
// frequent word on the same physical key must not replace that chosen reading.
func TestHsuPendingChoiceOverridesFrequency(t *testing.T) {
	for _, row := range []struct{ key, tone, reading, chosen, rival, rivalReading string }{
		{"a", "d", "ㄟ", "欸", "詞", "ㄘˊ"},
		{"e", "j", "ㄝ", "誒", "意", "ㄧˋ"},
		{"l", " ", "ㄥ", "鞥", "兒", "ㄦ"},
	} {
		t.Run(row.reading, func(t *testing.T) {
			chosenReading := zhuyin.Syllable(row.reading).WithTone(zhuyin.Syllable(row.rivalReading).ToneOf())
			s, _ := hsuSelectionSession(
				hsuSelectionWord{row.rival, row.rivalReading, .9},
				hsuSelectionWord{row.chosen, string(chosenReading), .001},
			)
			if snap := typeKeys(s, row.key+row.tone); snap.Text() != row.rival {
				t.Fatalf("test must first prefer the rival: %q", snap.Text())
			}
			s.Handle(proto.Request{Type: proto.Cancel})
			typeKeys(s, row.key)
			hsuChooseText(t, s, key(s, proto.KeyDown), row.reading)
			if snap := typeKeys(s, row.tone); snap.Text() != row.chosen {
				t.Fatalf("chosen %q was overridden by frequency: %q", row.reading, snap.Text())
			}
			if snap := key(s, proto.KeyEnter); snap.Commit != row.chosen {
				t.Fatalf("chosen reading lost on commit: %+v", snap)
			}
		})
	}
}

func TestHsuExplicitChoiceSurvivesEditingAndToggle(t *testing.T) {
	words := []hsuSelectionWord{
		{"詞", "ㄘˊ", .9}, {"欸", "ㄟˊ", .001}, {"我", "ㄨㄛˇ", .1},
		{"意", "ㄧˋ", .001}, {"誒", "ㄝˋ", .9},
	}
	t.Run("insert and delete before a chosen reading", func(t *testing.T) {
		s, _ := hsuSelectionSession(words...)
		key(s, "a")
		hsuChooseText(t, s, key(s, proto.KeyDown), "ㄟ")
		if snap := key(s, "d"); snap.Text() != "欸" {
			t.Fatalf("explicit reading: %q", snap.Text())
		}
		key(s, proto.KeyHome)
		if snap := typeKeys(s, "xhf"); snap.Text() != "我欸" {
			t.Fatalf("inserting before the choice changed its reading: %q", snap.Text())
		}
		key(s, proto.KeyHome)
		if snap := key(s, proto.KeyDelete); snap.Text() != "欸" {
			t.Fatalf("deleting before the choice changed its reading: %q", snap.Text())
		}
		if snap := key(s, proto.KeyDelete); snap.Text() != "" {
			t.Fatalf("could not delete the chosen syllable: %q", snap.Text())
		}
		// The deleted syllable's exact-reading restriction must not attach to
		// the next syllable. Here the alternate reading deliberately wins.
		if snap := typeKeys(s, "ej"); snap.Text() != "誒" {
			t.Fatalf("deleted choice leaked to a new syllable: %q", snap.Text())
		}
	})
	t.Run("immediate Shift toggle restores pending choice", func(t *testing.T) {
		s, _ := hsuSelectionSession(words...)
		key(s, "a")
		hsuChooseText(t, s, key(s, proto.KeyDown), "ㄟ")
		s.Handle(proto.Request{Type: proto.Toggle})
		s.Handle(proto.Request{Type: proto.Toggle, Restore: true})
		if snap := key(s, "d"); snap.Text() != "欸" {
			t.Fatalf("restored pending choice lost its exact reading: %q", snap.Text())
		}
	})
	t.Run("changing an unavailable tone retains the chosen reading", func(t *testing.T) {
		s, _ := hsuSelectionSession(
			hsuSelectionWord{"疵", "ㄘ", .9}, hsuSelectionWord{"欸", "ㄟ", .001},
		)
		key(s, "a")
		hsuChooseText(t, s, key(s, proto.KeyDown), "ㄟ")
		if snap := key(s, "d"); snap.Text() != "ㄟˊ" {
			t.Fatalf("missing reading should remain visible phonetics: %q", snap.Text())
		}
		key(s, proto.KeyBackspace)
		if snap := key(s, " "); snap.Text() != "欸" {
			t.Fatalf("changing the tone lost the chosen reading: %q", snap.Text())
		}
	})
}

func TestHsuNeutralToneKeepsReachableSharedReading(t *testing.T) {
	for _, row := range []struct{ key, rival, firstReading, chosen, neutralReading string }{
		{"a", "疵", "ㄘ", "欸", "ㄟ˙"},
		{"e", "衣", "ㄧ", "誒", "ㄝ˙"},
		{"l", "兒", "ㄦ", "鞥", "ㄥ˙"},
	} {
		t.Run(row.neutralReading, func(t *testing.T) {
			s, _ := hsuSelectionSession(
				hsuSelectionWord{row.rival, row.firstReading, .9},
				hsuSelectionWord{row.chosen, row.neutralReading, .001},
			)
			if snap := typeKeys(s, row.key+"s"); snap.Text() != row.chosen {
				t.Fatalf("reachable shared neutral reading fell back to first tone: %q", snap.Text())
			}
			if snap := key(s, proto.KeyEnter); snap.Commit != row.chosen {
				t.Fatalf("neutral reading lost on commit: %+v", snap)
			}
		})
	}
}

// A chosen single character still counts as a manual reading choice when the
// displayed path groups it inside a longer phrase, as it does on standard.
func TestHsuEnclosedSharedSelectionLearnsActualReading(t *testing.T) {
	for _, row := range []struct {
		layout layout.Layout
		keys   string
	}{{layout.Standard, "o6ji3"}, {layout.Hsu, "adxhf"}} {
		t.Run(row.layout.Name(), func(t *testing.T) {
			s, model := hsuSelectionSession(
				hsuSelectionWord{"詞", "ㄘˊ", .8}, hsuSelectionWord{"欸", "ㄟˊ", .001},
				hsuSelectionWord{"我", "ㄨㄛˇ", .1}, hsuSelectionWord{"欸我", "ㄟˊ ㄨㄛˇ", .9},
			)
			s.SetLayout(row.layout)
			typeKeys(s, row.keys)
			s.Handle(proto.Request{Type: proto.Cursor, To: 0})
			snap := s.Handle(proto.Request{Type: proto.Candidates, Length: 1})
			if snap = hsuChooseText(t, s, snap, "欸"); snap.Text() != "欸我" {
				t.Fatalf("selection should retain the longer phrase: %q", snap.Text())
			}
			if snap = key(s, proto.KeyEnter); snap.Commit != "欸我" {
				t.Fatalf("commit: %+v", snap)
			}
			if !model.Store.Readings["欸\tㄟˊ"] || !slices.Contains(model.Store.Picked["ㄟˊ"], "欸") {
				t.Fatalf("manual shared reading was not learned: readings=%v picked=%v", model.Store.Readings, model.Store.Picked)
			}
			if model.Store.Readings["欸\tㄘˊ"] || slices.Contains(model.Store.Picked["ㄘˊ"], "欸") {
				t.Fatal("learning used the physical key's other reading")
			}
		})
	}
}
