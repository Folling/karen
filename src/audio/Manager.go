package audio

import "sync"

var (
    instanceMutex sync.RWMutex
    instance *Manager
)

type Manager struct {
    connections map[string]*AudioPlayer
}

func GetManager() (*Manager) {
    instanceMutex.Lock()
    defer instanceMutex.Unlock()

    if instance == nil {
        instance = &Manager{}
    }

    return instance
}

func (m *Manager) JoinGuild(id string, player *AudioPlayer) {
    panic("stub!")
}

func (m* Manager) LeaveGuild(id string) {
    panic("stub!")
}

