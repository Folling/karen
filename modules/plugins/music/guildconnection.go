package music

import "sync"

// A connection to one guild's channel
type GuildConnection struct {
    sync.RWMutex

    // Controller channel for Skip/Pause/Resume
    controller chan controlMessage

    // Closer channel for Stop commands
    closer chan struct{}

    // Slice of waiting songs
    playlist []Song

    // Slice of waiting but unprocessed songs
    queue []Song

    // Whether this is playing music or not
    playing bool

    // A lock that stops the autoleaver while disconnecting
    leaveLock sync.RWMutex
}

// Helper to generate a guild connection
func (gc *GuildConnection) Alloc() *GuildConnection {
    gc.Lock()
    gc.playlist = []Song{}
    gc.queue = []Song{}
    gc.playing = false
    gc.Unlock()

    gc.CreateChannels()

    return gc
}

func (gc *GuildConnection) CloseChannels() {
    gc.Lock()
    close(gc.closer)
    close(gc.controller)
    gc.Unlock()
}

func (gc *GuildConnection) CreateChannels() {
    gc.Lock()
    gc.closer = make(chan struct{})
    gc.controller = make(chan controlMessage)
    gc.Unlock()
}

func (gc *GuildConnection) RecreateChannels() {
    gc.CloseChannels()
    gc.CreateChannels()
}
