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
			p := &RelativePosition{
				seconds:  tt.fields.seconds,
				captured: tt.fields.captured,
			}
			gotHour, gotMinute, gotSecond := p.GetPosition()
			if gotHour != tt.wantHour {
				t.Errorf("RelativePosition.GetPosition() gotHour = %v, want %v", gotHour, tt.wantHour)
			}
			if gotMinute != tt.wantMinute {
				t.Errorf("RelativePosition.GetPosition() gotMinute = %v, want %v", gotMinute, tt.wantMinute)
			}
			if gotSecond != tt.wantSecond && gotSecond != tt.wantSecond+1 {
				t.Errorf("RelativePosition.GetPosition() gotSecond = %v, want %v", gotSecond, tt.wantSecond)
			}
		})
	}
}

func TestNewRelativePosition(t *testing.T) {
	type args struct {
		hours   int
		minutes int
		seconds int
	}
	tests := []struct {
		name string
		args args
		want RelativePosition
	}{
		{"it should convert time into seconds", args{1, 22, 19}, RelativePosition{seconds: 4939}},
		{"it should convert time into seconds", args{0, 0, 0}, RelativePosition{seconds: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRelativePosition(tt.args.hours, tt.args.minutes, tt.args.seconds); got.seconds != tt.want.seconds {
				t.Errorf("NewRelativePosition() = %v, want %v", got, tt.want)
			}
		})
	}
}
