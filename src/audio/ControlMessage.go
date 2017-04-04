package audio

type ControlMessage uint8

const (
    Stop ControlMessage = iota
    Start
    Pause
    Resume
    Skip
)
