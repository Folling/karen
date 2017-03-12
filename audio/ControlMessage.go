package audio

type ControlMessage int

const (
    Stop ControlMessage = iota
    Start
    Pause
    Resume
    Skip
)
