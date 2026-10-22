package decode

import (
	"testing"

	"github.com/0c-biggo/0c-ime/core/input"
)

func TestZeroCostSharedReadingKeepsProvenance(t *testing.T) {
	d := fixture()
	d.Score.(uni)["應該"] = .5 // 讓未完成的第二音節也優先走共鍵讀音。
	lat := input.Lattice{{{Syl: "ㄧㄣ"}, {Syl: "ㄧㄥ"}}, {{Syl: "ㄍㄞ"}}}
	path := d.Decode(lat, nil, false)
	if len(path) != 1 || path[0].Word != "應該" || path[0].Matched != "ㄧㄥ ㄍㄞ" {
		t.Fatalf("zero-cost shared reading lost: %+v", path)
	}
	partial := d.Decode(lat[:1], nil, true)
	if len(partial) != 1 || partial[0].Word != "應" || !partial[0].Partial || partial[0].Matched != "ㄧㄥ" {
		t.Fatalf("partial shared reading lost: %+v", partial)
	}
}
