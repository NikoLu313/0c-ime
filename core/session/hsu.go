package session

import (
	"slices"

	"github.com/0c-biggo/0c-ime/core/input"
	"github.com/0c-biggo/0c-ime/core/layout"
	"github.com/0c-biggo/0c-ime/core/zhuyin"
)

// typedLattice 把許氏共鍵展開為成本相同的精確讀音。
// 字面標點不展開；標準／倚天仍使用原本的精確輸入格。
func (s *Session) typedLattice() input.Lattice {
	lat := s.front.Expand(s.syls)
	if s.layout != layout.Hsu {
		return lat
	}
	for i, hyps := range lat {
		if s.literals[i] || s.hsuExact[i] {
			continue
		}
		for _, h := range hyps {
			for _, syl := range layout.Alternatives(s.layout, h.Syl) {
				if !slices.ContainsFunc(lat[i], func(other input.Hyp) bool { return other.Syl == syl }) {
					lat[i] = append(lat[i], input.Hyp{Syl: syl, Cost: h.Cost})
				}
			}
		}
	}
	return lat
}

func (s *Session) shiftHsuExact(at, delta int) {
	if len(s.hsuExact) == 0 {
		return
	}
	next := map[int]bool{}
	for pos := range s.hsuExact {
		switch {
		case pos < at:
			next[pos] = true
		case delta > 0 || pos > at:
			next[pos+delta] = true
		}
	}
	s.hsuExact = next
}

// sharedReading 區分精確共鍵與近音修正；保留選字使用的真正讀音。
func (s *Session) sharedReading(start, end int, matched string) bool {
	if s.layout != layout.Hsu || matched == "" || start < 0 || end > len(s.syls) || start >= end {
		return false
	}
	reading := zhuyin.ParseReading(matched)
	if len(reading) != end-start {
		return false
	}
	changed := false
	for i, actual := range reading {
		typed := s.syls[start+i]
		if actual == typed {
			continue
		}
		if !slices.Contains(layout.Alternatives(s.layout, typed), actual) {
			return false
		}
		changed = true
	}
	return changed
}

func (s *Session) learningSyllables() []zhuyin.Syllable {
	if s.layout != layout.Hsu {
		return s.syls
	}
	syls := slices.Clone(s.syls)
	for _, g := range s.path {
		if s.sharedReading(g.Start, g.End, g.Matched) {
			copy(syls[g.Start:g.End], zhuyin.ParseReading(g.Matched))
		}
	}
	for at, lock := range s.locks {
		if s.sharedReading(at, at+lock.Len, lock.Matched) {
			copy(syls[at:at+lock.Len], zhuyin.ParseReading(lock.Matched))
		}
	}
	return syls
}

// openHsuReadings 讓獨立 ㄟ／ㄝ／ㄥ 也能先以 ↓ 選定注音，再輸入聲調。
// 這個清單選入的是組音狀態，並非無法繼續組音的字面符號。
func (s *Session) openHsuReadings() bool {
	if s.layout != layout.Hsu || s.comp.Empty() {
		return false
	}
	comp := s.comp
	sym, _ := layout.ResolveSymbol(s.layout, ' ', &comp)
	syl, _ := comp.Add(sym)
	alts := layout.Alternatives(s.layout, syl)
	if len(alts) == 0 {
		return false
	}
	items := []string{string(syl)}
	for _, alt := range alts {
		items = append(items, string(alt))
	}
	s.openSymbols(items, s.cursor, 0, true)
	s.cand.hsuReading = true
	return true
}
