package app

import (
	"reflect"
	"testing"
)

func TestTargetSeasonsNextSeasonOnly(t *testing.T) {
	got := targetSeasons(1, PrefetchConfig{SeasonsAhead: 2})
	want := []int32{2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targetSeasons() = %v, want %v", got, want)
	}
}

func TestTargetSeasonsCanIncludeCurrentSeason(t *testing.T) {
	got := targetSeasons(1, PrefetchConfig{SeasonsAhead: 2, IncludeCurrentSeason: true})
	want := []int32{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("targetSeasons() = %v, want %v", got, want)
	}
}
