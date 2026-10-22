package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 鍵盤排列預設標準；舊設定檔沒有欄位時不變；選倚天 41 鍵或許氏可保存、改回。
func TestKeyboardLayoutDefaultsStandardAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if s, err := Load(path); err != nil || s.KeyboardLayout != LayoutStandard {
		t.Fatalf("預設標準排列：%+v %v", s, err)
	}
	if err := os.WriteFile(path, []byte(`{"schema":1,"mixed_input":true,"future":42}`), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := Load(path); err != nil || s.KeyboardLayout != LayoutStandard {
		t.Fatalf("舊設定檔：%+v %v", s, err)
	}
	for _, value := range []string{"eten", "hsu", "standard", "eten", "hsu"} {
		s, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Set("keyboard-layout", value); err != nil {
			t.Fatal(err)
		}
		if err := Save(path, s); err != nil {
			t.Fatal(err)
		}
		back, err := Load(path)
		if err != nil || back.KeyboardLayout != value || !back.MixedInput {
			t.Fatalf("重新讀取：%+v %v", back, err)
		}
		b, _ := os.ReadFile(path)
		if !strings.Contains(string(b), `"keyboard_layout": "`+value+`"`) || !strings.Contains(string(b), `"future": 42`) {
			t.Fatalf("settings.json：%s", b)
		}
	}
	var s Settings
	if err := s.Set("keyboard-layout", "future-layout"); err == nil {
		t.Fatal("不認得的排列應拒絕")
	}
}

// 較新版本寫入本版不認得的排列：退回標準排列，存檔時原值保留。
func TestKeyboardLayoutUnknownValueKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	if err := os.WriteFile(path, []byte(`{"schema":1,"keyboard_layout":"future-layout"}`), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil || s.KeyboardLayout != LayoutStandard {
		t.Fatalf("讀不懂的排列用標準：%+v %v", s, err)
	}
	if err := s.Set("esc-clear", "off"); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"keyboard_layout":"future-layout"`) && !strings.Contains(string(b), `"keyboard_layout": "future-layout"`) {
		t.Fatalf("原值應保留：%s", b)
	}
}

func TestHsuDisablesMixedInputAndKeepsEmoji(t *testing.T) {
	s := Default()
	s.MixedInput = true
	if err := s.Set("keyboard-layout", LayoutHsu); err != nil {
		t.Fatal(err)
	}
	for _, it := range s.Items() {
		switch it.Key {
		case "keyboard-layout":
			if it.Value != LayoutHsu || it.ValueLabels[LayoutHsu] != "許氏注音" || !strings.Contains(strings.Join(it.Values, "|"), LayoutHsu) {
				t.Fatalf("許氏應能在設定介面選取：%+v", it)
			}
		case "mixed-input":
			if !it.Disabled || it.DisabledReason != "許氏注音時關閉" || it.Value != "on" || it.Effective != "off" {
				t.Fatalf("許氏停用混打並保留原值：%+v", it)
			}
		case "emoji-shortcut", "emoji-equals-first":
			if it.Disabled {
				t.Fatalf("許氏的 = 不與注音衝突，Emoji 應保留：%+v", it)
			}
		}
	}
	if s.EffectiveMixedInput() || !s.MixedInput || s.EmojiShortcut != "equals" {
		t.Fatalf("混打實際關閉、原設定與 Emoji 保留：%+v", s)
	}
	for _, name := range []string{LayoutStandard, LayoutETen} {
		if err := s.Set("keyboard-layout", name); err != nil {
			t.Fatal(err)
		}
		if !s.EffectiveMixedInput() {
			t.Fatalf("%s 應沿用既有的引擎設定", name)
		}
	}
	if err := s.Set("mixed-input", "off"); err != nil {
		t.Fatal(err)
	}
	if s.EffectiveMixedInput() {
		t.Fatal("標準排列也應尊重關閉混打的設定")
	}
}

// 倚天 41 鍵時，中英混打與 Emoji 選單（含子選項）標為停用並附原因；值保留，切回標準排列就照舊可改。
func TestETenDisablesMixedAndEmojiItems(t *testing.T) {
	s := Default()
	s.MixedInput = true
	s.Set("toggle-key", ToggleRight) // macOS 預設不用 Shift，開始模式另外停用（#184）；這裡只看倚天
	disabled := func(s Settings) map[string]string {
		out := map[string]string{}
		for _, it := range s.Items() {
			if it.Disabled {
				out[it.Key] = it.DisabledReason
			}
		}
		return out
	}
	if d := disabled(s); len(d) != 0 {
		t.Fatalf("標準排列沒有停用的項目：%v", d)
	}
	if err := s.Set("keyboard-layout", "eten"); err != nil {
		t.Fatal(err)
	}
	d := disabled(s)
	if len(d) != 3 || d["mixed-input"] != "倚天 41 鍵時關閉" || d["emoji-shortcut"] != "倚天 41 鍵時關閉" || d["emoji-equals-first"] == "" {
		t.Fatalf("倚天停用混打與 Emoji：%v", d)
	}
	for _, it := range s.Items() {
		if it.Key == "mixed-input" && it.Value != "on" || it.Key == "emoji-shortcut" && it.Value != "equals" {
			t.Fatalf("停用時仍回報設定檔中的值：%+v", it)
		}
	}
	if err := s.Set("keyboard-layout", "standard"); err != nil {
		t.Fatal(err)
	}
	if d := disabled(s); len(d) != 0 || !s.MixedInput || s.EmojiShortcut != "equals" {
		t.Fatalf("切回標準排列恢復原本的值：%v %+v", d, s)
	}
}
