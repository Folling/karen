package music

import (
    "time"
    "encoding/binary"
    "io"
    "github.com/bwmarrin/discordgo"
    "os"
    "git.lukas.moe/sn0w/Karen/helpers"
)

// startPlayer is a helper to call play()
func (m *Module) startPlayer(guild string, vc *discordgo.VoiceConnection, msg *discordgo.Message, session *discordgo.Session) {
    defer helpers.RecoverDiscord(msg)

    // Ignore call if already playing
    if m.guildConnections[guild].playing {
        return
    }

    // Get pointer to closer and controller via guildConnection
    closer := &m.guildConnections[guild].closer
    controller := &m.guildConnections[guild].controller
    playlist := &m.guildConnections[guild].playlist

    // Start eventloop
    for {
        // Exit if the closer channel closes
        select {
        case <-(*closer):
            return
        default:
        }

        // Do nothing until voice is ready and songs are queued
        if !vc.Ready || len(*playlist) == 0 {
            time.Sleep(1 * time.Second)
            continue
        }

        // Mark guild as playing
        m.guildConnections[guild].Lock()
        m.guildConnections[guild].playing = true
        m.guildConnections[guild].Unlock()

        // Send data to discord
        // Blocks until the song is done
        m.play(vc, *closer, *controller, (*playlist)[0], msg, session)

        // Remove song from playlist if it's not empty
        if len(*playlist) > 0 {
            m.guildConnections[guild].Lock()
            *playlist = append((*playlist)[:0], (*playlist)[1:]...)
            m.guildConnections[guild].Unlock()
        }
    }
}

// play is responsible for streaming the OPUS data to discord
func (m *Module) play(
    vc *discordgo.VoiceConnection,
    closer <-chan struct{},
    controller <-chan controlMessage,
    song Song,
    msg *discordgo.Message,
    session *discordgo.Session,
) {
    // Mark as speaking
    vc.Speaking(true)

    // Mark as not speaking as soon as we're done
    defer vc.Speaking(false)

    // Read file
    file, err := os.Open(song.Path)
    helpers.Relax(err)
    defer file.Close()

    // Allocate opus header buffer
    var opusLength int16

    // Start eventloop
    for {
        // Exit if the closer channel closes
        select {
        case <-closer:
            return
        default:
        }

        // Listen for commands from controller
        select {
        case ctl := <-controller:
            switch ctl {
            case Skip:
                return
            case Pause:
                wait := true
                iteration := 0
                session.ChannelMessageSend(msg.ChannelID, ":pause_button: Track paused")
                for {
                    // Read from controller channel
                    ctl := <-controller
                    switch ctl {
                    case Skip:
                        return
                    case Resume:
                        wait = false
                    }

                    // If Skip or Resume was received end lock
                    if !wait {
                        break
                    }

                    // Sleep for 0.5s until next check to reduce CPU load
                    iteration++
                    time.Sleep(500 * time.Millisecond)
                }
                session.ChannelMessageSend(msg.ChannelID, ":play_pause: Track resumed")
            default:
            }
        default:
        }

        // Read opus frame length
        err = binary.Read(file, binary.LittleEndian, &opusLength)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            return
        }
        helpers.Relax(err)

        // Read audio data
        opus := make([]byte, opusLength)
        err = binary.Read(file, binary.LittleEndian, &opus)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            return
        }
        helpers.Relax(err)

        // Send to discord
        vc.OpusSend <- opus
    }
}
