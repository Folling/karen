package audio

import "sync"

var (
    instanceMutex sync.RWMutex
    instance *AudioManger
)

type AudioManger struct {
    connections map[string]*AudioPlayer
}

func GetManager() (*AudioManger) {
    instanceMutex.Lock()
    defer instanceMutex.Unlock()

    if instance == nil {
        instance = &AudioManger{}
    }

    return instance
}

func (m *AudioManger) JoinGuild(id string, player *AudioPlayer) {
    panic("stub!")
}

func (m* AudioManger) LeaveGuild(id string) {
    panic("stub!")
}

