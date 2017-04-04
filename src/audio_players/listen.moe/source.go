package audio_players

import "sync"

type ListenDotMoeSource struct {
    sync.RWMutex

    ready bool
}

func (l *ListenDotMoeSource) New() (error) {

}

func (l *ListenDotMoeSource) Free() (error) {

}

func (l *ListenDotMoeSource) Provide() (chan *[]byte, error) {

}

func (l *ListenDotMoeSource) Ready() (bool) {
    return l.ready
}
