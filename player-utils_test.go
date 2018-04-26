package main

import (
	"testing"
	"time"
)

func TestRelativePosition_GetPosition(t *testing.T) {
	type fields struct {
		seconds  int
		captured time.Time
	}
	tests := []struct {
		name       string
		fields     fields
		wantHour   int
		wantMinute int
		wantSecond int
	}{
		{"split hour, minute, seconds is right", fields{4939, time.Now()}, 1, 22, 19},
		{"split hour, minute, seconds is right with delta", fields{4939, time.Now().Add(- 40*time.Minute - 51*time.Second)}, 2, 3, 10},
		{"split is 0 safe", fields{0, time.Now()}, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &TimePosition{
				seconds:  tt.fields.seconds,
				captured: tt.fields.captured,
			}
			pos := p.GetPosition()
			if pos.Hours != tt.wantHour {
				t.Errorf("TimePosition.GetPosition() gotHour = %v, want %v", pos.Hours, tt.wantHour)
			}
			if pos.Minutes != tt.wantMinute {
				t.Errorf("TimePosition.GetPosition() gotMinute = %v, want %v", pos.Minutes, tt.wantMinute)
			}
			if pos.Seconds != tt.wantSecond && pos.Seconds != tt.wantSecond+1 {
				t.Errorf("TimePosition.GetPosition() gotSecond = %v, want %v", pos.Seconds, tt.wantSecond)
			}
		})
	}
}

func TestNewRelativePosition(t *testing.T) {
	type args struct {
		hours    int
		minutes  int
		seconds  int
		absolute bool
	}
	tests := []struct {
		name string
		args args
		want TimePosition
	}{
		{"it should convert time into seconds", args{1, 22, 19, true}, TimePosition{seconds: 4939}},
		{"it should convert time into seconds", args{0, 0, 0, true}, TimePosition{seconds: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTimePosition(tt.args.hours, tt.args.minutes, tt.args.seconds, tt.args.absolute); got.seconds != tt.want.seconds {
				t.Errorf("NewTimePosition() = %v, want %v", got, tt.want)
			}
		})
	}
}
