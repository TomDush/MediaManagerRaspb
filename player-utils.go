package main

import "time"

// Store position in a media, considering time between update and request
type RelativePosition struct {
	seconds  int
	captured time.Time
}

// get current position based on last update
func (p *RelativePosition) GetPosition() (hour int, minute int, second int) {
	delta := time.Now().Sub(p.captured).Seconds()
	secs := p.seconds + int(delta)

	return secs / 3600, (secs % 3600) / 60, secs % 60
}

// Create Relative position, captured at this instant
func NewRelativePosition(hours int, minutes int, seconds int) RelativePosition {
	return RelativePosition{
		seconds:  hours*3600 + minutes*60 + seconds,
		captured: time.Now(),
	}
}

// Status DTO, built for REST API
type PlayerStatus struct {
	Playing bool    `json:"playing"`
	Media   FileDto `json:"media"`
	Position struct {
		Hours   int `json:"hours"`
		Minutes int `json:"minutes"`
		Seconds int `json:"seconds"`
	} `json:"position"`
}

// Status when playing
func NewPlayerStatus(media File, position RelativePosition) PlayerStatus {
	st := PlayerStatus{
		Playing: true,
		Media:   NewFileDto(media),
	}

	st.Position.Hours, st.Position.Minutes, st.Position.Seconds = position.GetPosition()

	return st
}

// Status when not playing
func NotPlayingStatus() PlayerStatus {
	return PlayerStatus{Playing: false}
}
