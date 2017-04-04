package audio

type AudioSource interface {
    // New allocates and initiates the audio source.
    // This might include connecting to a webradio or downloading a song.
    // The AudioPlayer will wait until Ready() returns true (i.e the source is able to Provide() Data).
    // Note: If your source needs any kind of transcoding/encoding you should process as much as possible in advance
    // during the execution of this function.
    New() (error)

    // Free is expected to deallocate any in-use resources and stop all child-routines (if needed)
    // This should also close the Provide() chan and set Ready() to false.
    Free() (error)

    // Provider has to return a chan that transfers pointers to the source's opus frames.
    // Again: While live transcoding/encoding is fully allowed and possible, it is recommended to
    // do the heavy lifting in New() to avoid lags.
    Provide() (chan *[]byte, error)

    // Ready indicates if the AudioSource is ready to Provide() data.
    Ready() (bool)
}
