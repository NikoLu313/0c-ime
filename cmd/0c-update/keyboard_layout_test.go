package main

import (
	"strings"
	"testing"

	"github.com/0c-biggo/0c-ime/internal/settings"
)

// #83：鍵盤排列經 settings --json 給設定程式；選倚天 41 鍵時，中英混打與 Emoji 選單標為停用並附原因，
// 值仍是設定檔中的值，與引擎載入的值比較待套用狀態。
func TestKeyboardLayoutContract(t *testing.T) {
	base := t.TempDir()
	writeSettings(t, base, `{"keyboard_layout":"eten","mixed_input":true}`)
	if v, ok := runningValues(map[string]string{"keyboard_layout": "standard"})["keyboard-layout"]; !ok || v != "standard" {
		t.Fatalf("health 的 keyboard_layout：%q %v", v, ok)
	}
	doc := buildSettingsDoc(base, modelStatusDial(map[string]string{"keyboard_layout": "standard", "mixed_input": "true", "emoji_shortcut": "equals"}, true))
	layout := item(t, doc, "keyboard-layout")
	if layout.Value != "eten" || !layout.Pending || layout.Disabled || !doc.PendingRestart {
		t.Fatalf("排列已儲存、引擎尚未套用：%+v", layout)
	}
	for _, key := range []string{"mixed-input", "emoji-shortcut", "emoji-equals-first"} {
		it := item(t, doc, key)
		if !it.Disabled || it.DisabledReason != "倚天 41 鍵時關閉" || it.Pending {
			t.Fatalf("%s 應停用、不因停用而顯示待套用：%+v", key, it)
		}
	}
	if it := item(t, doc, "mixed-input"); it.Value != "on" {
		t.Fatalf("停用時保留使用者的值：%+v", it)
	}
	if it := item(t, doc, "space-selects"); it.Disabled {
		t.Fatalf("其他設定不受影響：%+v", it)
	}
	if o, err := parseSetArgs([]string{"keyboard-layout", "eten", "-apply", "-json"}); err != nil || len(o.pairs) != 1 || !o.apply {
		t.Fatalf("%+v %v", o, err)
	}
}

func TestHsuKeyboardLayoutContract(t *testing.T) {
	base := t.TempDir()
	path := writeSettings(t, base, `{"keyboard_layout":"standard","mixed_input":true,"future":42}`)
	o, err := parseSetArgs([]string{"keyboard-layout", "hsu", "-json"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := runSet(base, noEngine, o, nop)
	if err != nil || !res.OK {
		t.Fatalf("儲存許氏排列：%+v %v", res, err)
	}
	s, err := settings.Load(path)
	if err != nil || s.KeyboardLayout != settings.LayoutHsu || !s.MixedInput || !strings.Contains(readFile(t, path), `"future": 42`) {
		t.Fatalf("許氏設定與未知欄位應能保存：%+v %v", s, err)
	}
	doc := buildSettingsDoc(base, modelStatusDial(map[string]string{"keyboard_layout": "hsu", "mixed_input": "false", "emoji_shortcut": "equals", "emoji_equals_first": "true"}, true))
	if it := item(t, doc, "keyboard-layout"); it.Value != "hsu" || it.Pending || it.ValueLabels["hsu"] != "許氏注音" {
		t.Fatalf("許氏已套用：%+v", it)
	}
	if it := item(t, doc, "mixed-input"); !it.Disabled || it.Value != "on" || it.Effective != "off" || it.Pending {
		t.Fatalf("混打原值保留，實際已關閉，不應持續顯示待套用：%+v", it)
	}
	for _, key := range []string{"emoji-shortcut", "emoji-equals-first"} {
		if it := item(t, doc, key); it.Disabled || it.Pending {
			t.Fatalf("許氏應能使用 Emoji：%+v", it)
		}
	}
	if doc.PendingRestart {
		t.Fatal("已套用許氏不應顯示待重新啟動")
	}
}
