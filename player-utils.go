package main

import (
	"time"
)

// Store position in a media, considering time between update and request
type TimePosition struct {
	seconds  int
	absolute bool
	captured time.Time
}

// get current position based on last update
func (p *TimePosition) GetPosition() *TimePositionDto {
	secs := p.GetSeconds()
	return NewTimePositionDto(secs/3600, (secs%3600)/60, secs%60)
}

func (p *TimePosition) GetSeconds() int {
	secs := p.seconds
	if !p.absolute {
		delta := time.Now().Sub(p.captured).Seconds()
		secs += int(delta)
	}
	return secs
}

// Create a absolute or not absolute version of this time position
func (p TimePosition) Absolute(absolute bool) TimePosition {
	return TimePosition{
		p.GetSeconds(),
		absolute,
		time.Now(),
	}
}

// Create Relative position, captured at this instant
func NewTimePosition(hours int, minutes int, seconds int, absolute bool) TimePosition {
	return TimePosition{
		hours*3600 + minutes*60 + seconds,
		absolute,
		time.Now(),
	}
}

type TimePositionDto struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
	Seconds int `json:"seconds"`
}

func NewTimePositionDto(hours int, minutes int, seconds int) *TimePositionDto {
	return &TimePositionDto{
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}
}

// Status DTO, built for REST API
type PlayerStatus struct {
	Playing  bool             `json:"playing"`
	Paused   bool             `json:"paused"`
	Media    *FileDto         `json:"media"`
	Position *TimePositionDto `json:"position"`
	Length   *TimePositionDto `json:"length"`
}

// Status when playing
func NewPlayerStatus(media File, paused bool, position TimePosition, length TimePosition) PlayerStatus {
	mediaDto := NewFileDto(media)
	return PlayerStatus{
		Playing:  true,
		Paused:   paused,
		Media:    &mediaDto,
		Position: position.GetPosition(),
		Length:   length.GetPosition(),
	}
}

// Status when not playing
func NotPlayingStatus() PlayerStatus {
	return PlayerStatus{Playing: false}
}
