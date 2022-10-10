package server

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/rs/zerolog/log"
)

type PlaybackEvent struct {
	status *string
	text   *string
}

type spotifyWatcher struct {
	conn *dbus.Conn
	ch   chan *dbus.Signal
	out  chan PlaybackEvent
}

func newSpotifyWatcher() (*spotifyWatcher, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect dbus: %v", err)
	}

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/mpris/MediaPlayer2"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchSender("org.mpris.MediaPlayer2.spotify"),
	); err != nil {
		conn.Close()
		return nil, fmt.Errorf("dbus add match signal: %v", err)
	}

	ch := make(chan *dbus.Signal, 10)
	out := make(chan PlaybackEvent)
	s := spotifyWatcher{conn, ch, out}

	conn.Signal(ch)
	go s.loop()

	return &s, nil
}

func (s *spotifyWatcher) loop() {
	var last *PlaybackEvent
	for msg := range s.ch {
		if msg.Body[0].(string) != "org.mpris.MediaPlayer2.Player" {
			continue
		}
		body := msg.Body[1].(map[string]dbus.Variant)
		event := PlaybackEvent{}
		if value, ok := body["PlaybackStatus"]; ok {
			status := value.Value().(string)
			event.status = &status
		}
		if value, ok := body["Metadata"]; ok {
			metadata := value.Value().(map[string]dbus.Variant)
			artists := metadata["xesam:artist"].Value().([]string)
			title := metadata["xesam:title"].Value().(string)
			text := strings.Join(artists, " & ") + " - " + title
			event.text = &text
		}
		if last == nil || event != *last {
			last = &event
			s.out <- event
		}
	}
}

func (s *spotifyWatcher) close() {
	close(s.ch)
	close(s.out)

	err := s.conn.Close()
	if err != nil {
		log.Error().Err(err).Msg("spotify dbus close")
	}
}
