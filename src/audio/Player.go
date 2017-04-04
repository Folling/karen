package audio

// The AudioPlayer delegates AudioSource streams to OpusChan's of discordgo.
// It is expected that players implement a FILO-stack based playlist that is managed internally and only exposed
// through PushSource() and PopSource().
type Player interface {
    // New initiates the AudioPlayer.
    // It takes a channel of an established discord connection as first argument.
    New(discordChan chan []byte) (error)

    // Control takes and interprets a ControlMessage
    Control(message ControlMessage)

    // GetSource returns the currently active source
    GetSource() (*Source, error)

    // PushSource adds a new AudioSource to this player's playlist
    PushSource(source *Source) (error)

    // PopSource removes the first item from this playlist
    PopSource() (*Source, error)

    // SetSource empties the current playlist
    ClearSources() (error)
}
