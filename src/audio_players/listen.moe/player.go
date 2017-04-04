package audio_players

import "code.lukas.moe/x/karen/src/audio"

type ListenDotMoePlayer struct {

}

func (l *ListenDotMoePlayer) New(discordChan chan []byte) (error) {

}

func (l *ListenDotMoePlayer) Control(message audio.ControlMessage) {

}

func (l *ListenDotMoePlayer) GetSource() (*audio.Source, error) {

}

func (l *ListenDotMoePlayer) PushSource(source *audio.Source) (error) {

}

func (l *ListenDotMoePlayer) PopSource() (*audio.Source, error) {

}

func (l *ListenDotMoePlayer) ClearSources() (error) {

}
