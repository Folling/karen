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
    if instance == nil {
        instanceMutex.Lock()
        instance = &AudioManger{}
        instanceMutex.Unlock()
    }

    return instance
}

func (m *AudioManger) JoinGuild(id string, player *AudioPlayer) {
    panic("stub!")
}

func (m* AudioManger) LeaveGuild(id string) {
    panic("stub!")
}

