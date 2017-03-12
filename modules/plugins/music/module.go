package music

import (
    "strings"
    "github.com/bwmarrin/discordgo"
    "os"
    "regexp"
    "fmt"
    "strconv"
    "os/exec"
    "github.com/Jeffail/gabs"
    Logger "git.lukas.moe/sn0w/Karen/logger"
    "time"
    "io/ioutil"
    "git.lukas.moe/sn0w/Karen/cache"
    "git.lukas.moe/sn0w/Karen/helpers"
    rethink "github.com/gorethink/gorethink"
)

type Module struct {
    guildConnections map[string]*GuildConnection
    enabled          bool
}

func (m *Module) Commands() []string {
    return []string{
        "join",
        "leave",

        "play",
        "pause",
        "stop",
        "skip",
        "next",
        "playing",
        "np",

        "add",
        "list",
        "playlist",
        "random",
        "rand",
        "search",
        "find",

        "mdev",
    }
}

func (m *Module) Init(session *discordgo.Session) {
    m.enabled = false
    foundYTD, foundFFPROBE, foundFFMPEG, foundRopus := false, false, false, false

    Logger.PLUGIN.L("music", "Checking for youtube-dl, ropus, ffmpeg and ffprobe...")
    for _, path := range strings.Split(os.Getenv("PATH"), ":") {
        files, _ := ioutil.ReadDir(path)

        for _, file := range files {
            switch {
            case regexp.MustCompile(`youtube-dl`).Match([]byte(file.Name())):
                foundYTD = true
                break

            case regexp.MustCompile(`ffprobe`).Match([]byte(file.Name())):
                foundFFPROBE = true
                break

            case regexp.MustCompile(`ffmpeg`).Match([]byte(file.Name())):
                foundFFMPEG = true
                break

            case regexp.MustCompile(`ropus`).Match([]byte(file.Name())):
                foundRopus = true
                break
            }
        }
    }

    if foundYTD && foundFFPROBE && foundFFMPEG && foundRopus {
        m.enabled = true
        Logger.PLUGIN.L("music", "Found. Music enabled!")

        // Allocate connections map
        m.guildConnections = make(map[string]*GuildConnection)

        // Start loop that processes videos in background
        Logger.PLUGIN.L("music", "Starting async processor loop")
        go processorLoop()

        // Start janitor that removes files which are not tracked in the DB
        Logger.PLUGIN.L("music", "Starting async janitor")
        go janitor()
    } else {
        Logger.PLUGIN.L("music", "Not Found. Music disabled!")
    }
}

func (m *Module) Action(command string, content string, msg *discordgo.Message, session *discordgo.Session) {
    // Only continue if enabled
    if !m.enabled {
        return
    }

    // Store channel ref
    channel, err := cache.Channel(msg.ChannelID)
    helpers.Relax(err)

    // Only continue if the voice is available
    if !helpers.VoiceIsFreeOrOccupiedBy(channel.GuildID, "music") {
        helpers.VoiceSendStatus(channel.ID, channel.GuildID, session)
        return
    }

    // Store guild ref
    guild, err := session.Guild(channel.GuildID)
    helpers.Relax(err)

    // Store voice channel ref
    vc := m.resolveVoiceChannel(msg.Author, guild, session)

    // Store voice connection ref (deferred)
    var voiceConnection *discordgo.VoiceConnection

    // Check if the user is connected to voice at all
    if vc == nil {
        session.ChannelMessageSend(channel.ID, "You're either not in the voice-chat or I can't see you :neutral_face:")
        return
    }

    // He is connected for sure.
    // The routine would've stopped otherwise
    // Check if we are present in this channel too
    if session.VoiceConnections[guild.ID] == nil || session.VoiceConnections[guild.ID].ChannelID != vc.ID {
        // Nope.
        // Check if the user wanted us to join.
        // Else report the error
        if command == "join" {
            helpers.VoiceOccupy(guild.ID, "music")

            message, merr := session.ChannelMessageSend(channel.ID, ":arrows_counterclockwise: Joining...")

            voiceConnection, err = session.ChannelVoiceJoin(guild.ID, vc.ID, false, false)
            if err != nil {
                helpers.VoiceFree(guild.ID)
                helpers.Relax(err)
            }

            if merr == nil {
                session.ChannelMessageEdit(channel.ID, message.ID, "Joined! :smiley:")

                // Make guild connection
                if m.guildConnections[guild.ID] == nil {
                    m.guildConnections[guild.ID] = (&GuildConnection{}).Alloc()
                }

                // Start auto-leaver
                go m.autoLeave(guild.ID, channel.ID, session, m.guildConnections[guild.ID].closer)

                return
            }

            helpers.Relax(merr)
        } else {
            session.ChannelMessageSend(channel.ID, "You should join the channel I'm in or make me join yours before telling me to do stuff :thinking:")
        }

        return
    }

    // We are connected \o/
    // Save the ref for easier access
    voiceConnection = session.VoiceConnections[guild.ID]

    // Store gc pointer for easier access
    queue := &m.guildConnections[guild.ID].queue
    playlist := &m.guildConnections[guild.ID].playlist

    // Dev command that shows the guild connection
    if command == "mdev" {
        session.ChannelMessageSend(
            channel.ID,
            fmt.Sprintf("```\n%#v\n```", m.guildConnections[guild.ID]),
        )
        return
    }

    // Check what the user wants from us
    switch command {
    case "leave":
        session.ChannelTyping(channel.ID)

        m.guildConnections[guild.ID].leaveLock.RLock()
        voiceConnection.Disconnect()
        m.guildConnections[guild.ID].CloseChannels()

        delete(m.guildConnections, guild.ID)
        helpers.VoiceFree(guild.ID)

        session.ChannelMessageSend(channel.ID, "OK, bye :frowning:")
        break

    case "play":
        if len(*playlist) == 0 {
            if len(*queue) > 0 {
                session.ChannelMessageSend(channel.ID, "Please wait until your downloads are finished :wink:")
                return
            }

            session.ChannelMessageSend(channel.ID, "You should add some music first :thinking:")
            return
        }

        if m.guildConnections[guild.ID].playing {
            m.guildConnections[guild.ID].controller <- Resume
        } else {
            go m.startPlayer(guild.ID, voiceConnection, msg, session)
        }
        break

    case "pause":
        m.guildConnections[guild.ID].controller <- Pause
        break

    case "stop":
        session.ChannelTyping(channel.ID)

        m.guildConnections[guild.ID].RecreateChannels()

        m.guildConnections[guild.ID].Lock()
        m.guildConnections[guild.ID].playing = false
        *playlist = []Song{}
        m.guildConnections[guild.ID].Unlock()

        session.ChannelMessageSend(channel.ID, ":stop_button: Playback stopped. (Playlist is now empty)")
        break

    case "skip", "next":
        m.guildConnections[guild.ID].controller <- Skip
        break

    case "playing", "np":
        session.ChannelTyping(channel.ID)

        if !m.guildConnections[guild.ID].playing {
            session.ChannelMessageSend(
                channel.ID,
                "Why do you ask me? You didn't even start playback yet!",
            )
            return
        }

        if len(*playlist) == 0 {
            session.ChannelMessageSend(
                channel.ID,
                "Seems like you cleared the Playlist. \n I don't know what's playing :neutral_face:",
            )
            return
        }

        session.ChannelMessageSend(
            channel.ID,
            ":musical_note: Currently playing `"+(*playlist)[0].Title+"`",
        )
        break

    case "list", "playlist":
        session.ChannelTyping(channel.ID)

        if len(*playlist) == 0 {
            session.ChannelMessageSend(channel.ID, "Playlist is empty ¯\\_(ツ)_/¯")
            return
        }

        embed := &discordgo.MessageEmbed{
            Title: ":musical_note: Playlist",
            Color: 0x0FADED,
            Fields: []*discordgo.MessageEmbedField{
                {Name: "Current", Value: "", Inline: false},
                {Name: "Queue", Value: "", Inline: false},
            },
            Footer: &discordgo.MessageEmbedFooter{
                Text: "",
            },
        }

        for i, song := range *playlist {
            if i > 11 {
                embed.Footer.Text = strconv.Itoa(len(*playlist)-i) + " entries omitted."
                break
            }

            num := "`[" + strconv.Itoa(i+1) + "]` "

            if i == 0 {
                embed.Fields[0].Value = num + " " + song.Title
                continue
            }

            embed.Fields[1].Value += num + " " + song.Title + "\n"
        }

        // Catch an empty queue
        if embed.Fields[1].Value == "" {
            embed.Fields[1].Value = "Empty ¯\\_(ツ)_/¯"
        }

        session.ChannelMessageSendEmbed(channel.ID, embed)
        break

    case "random", "rand":
        session.ChannelTyping(channel.ID)

        cursor, err := rethink.Table("music").Filter(map[string]interface{}{"processed": true}).Run(helpers.GetDB())
        helpers.Relax(err)
        defer cursor.Close()

        var matches []Song
        err = cursor.All(&matches)
        helpers.Relax(err)

        var match Song

        for _, song := range matches {
            dupe := false

            for _, entry := range *playlist {
                if song.ID == entry.ID {
                    dupe = true
                    break
                }
            }

            if !dupe {
                match = song
                break
            }
        }

        if match == (Song{}) {
            session.ChannelMessageSend(channel.ID, "Sorry but there are no more songs that aren't already in your playlist :shrug:")
            return
        }

        m.guildConnections[guild.ID].Lock()
        *playlist = append(*playlist, match)
        m.guildConnections[guild.ID].Unlock()

        session.ChannelMessageSend(channel.ID, ":ballot_box_with_check: Added `"+match.Title+"`")
        break

    case "add":
        session.ChannelTyping(channel.ID)

        content = strings.TrimSpace(content)
        if content == "" {
            _, err := session.ChannelMessageSend(channel.ID, "You have to send me a link :neutral_face:")
            helpers.Relax(err)
            return
        }

        // Trim masquerade if present
        contentRune := []rune(content)
        if contentRune[0] == '<' && contentRune[len(contentRune)-1] == '>' {
            content = string(contentRune[1: len(contentRune)-1])
        }

        // Check if the url is a playlist
        if strings.Contains(content, "list") ||
            strings.Contains(content, "/set") ||
            strings.Contains(content, "/mix") {
            session.ChannelMessageSend(channel.ID, "Sorry but playlists are not supported :neutral_face:")
            return
        }

        // Check if the link has been cached
        cursor, err := rethink.Table("music").Filter(map[string]interface{}{"url": content}).Run(helpers.GetDB())
        helpers.Relax(err)
        defer cursor.Close()

        var match Song
        err = cursor.One(&match)

        // Check if the song does not yet exists in the DB
        if err == rethink.ErrEmptyResult {
            // Link was not downloaded yet

            // Resolve the url through YTDL.
            ytdl := exec.Command("youtube-dl", "-J", "--flat-playlist", content)
            yout, yerr := ytdl.Output()

            // If youtube-dl exits with 0 the link is valid
            if yerr != nil {
                session.ChannelMessageSend(channel.ID, "That looks like an invalid or unspported download link :frowning:")
                return
            }

            // Parse info JSON and allocate song object
            json, err := gabs.ParseJSON(yout)
            helpers.Relax(err)

            // Exit if the link is a live stream
            // (archived live streams are allowed though)
            if json.ExistsP("is_live") && json.Path("is_live").Data().(bool) {
                session.ChannelMessageSend(channel.ID, "Livestreams are not supported :neutral_face:")
                return
            }

            // Exit if the link is a playlist
            if json.ExistsP("_type") && json.Path("_type").Data() != nil {
                switch json.Path("_type").Data().(string) {
                case "playlist":
                    session.ChannelMessageSend(channel.ID, "Sorry but playlists are not supported :neutral_face:")
                    return
                }
            }

            match = Song{
                AddedBy:   msg.Author.ID,
                Processed: false,
                Title:     json.Path("title").Data().(string),
                Duration:  int(json.Path("duration").Data().(float64)),
                URL:       content,
            }

            // Check if the video is not too long
            // Bot owners may bypass this
            if !helpers.IsBotAdmin(msg.Author.ID) && match.Duration > int((65 * time.Minute).Seconds()) {
                session.ChannelMessageSend(channel.ID, "Whoa `"+match.Title+"` is a big video!\nPlease use something shorter :neutral_face:")
                return
            }

            // Add to db
            _, e := rethink.Table("music").Insert(match).RunWrite(helpers.GetDB())
            helpers.Relax(e)

            // Add to queue
            m.guildConnections[guild.ID].Lock()
            *queue = append(*queue, match)
            m.guildConnections[guild.ID].Unlock()

            // Inform users
            session.ChannelMessageSend(
                channel.ID,
                "`"+match.Title+"` was added to your download-queue.\nLive progress at: <https://karen.vc/music>",
            )
            go m.waitForSong(channel.ID, guild.ID, match, msg, session)
            return
        } else if err != nil {
            // Unknown error. Should report that.
            helpers.Relax(err)
            return
        }

        // Song present in DB
        // Check if the match was already processed
        if match.Processed {
            // Was processed. Add to playlist.
            m.guildConnections[guild.ID].Lock()
            *playlist = append(*playlist, match)
            m.guildConnections[guild.ID].Unlock()

            session.ChannelMessageSend(channel.ID, "Added from cache! :ok_hand:")
            return
        }

        // Not yet processed. Check if it's in the queue
        songPresent := false
        for _, song := range *queue {
            if match.URL == song.URL {
                songPresent = true
                break
            }
        }

        if songPresent {
            session.ChannelMessageSend(
                channel.ID,
                "`"+match.Title+"` is already in your download-queue.\nLive progress at: <https://karen.vc/music>",
            )
            return
        }

        m.guildConnections[guild.ID].Lock()
        *queue = append(*queue, match)
        m.guildConnections[guild.ID].Unlock()

        session.ChannelMessageSend(
            channel.ID,
            "`"+match.Title+"` was added to your download-queue.\nLive progress at: <https://karen.vc/music>",
        )
        go m.waitForSong(channel.ID, guild.ID, match, msg, session)
        break

    case "search", "find":
        session.ChannelTyping(channel.ID)

        if len(content) < 4 {
            session.ChannelMessageSend(
                channel.ID,
                "Kinda short search term :thinking:\nTry something longer!",
            )
            return
        }

        term := "(?i).*" + strings.Join(strings.Split(content, " "), ".*") + ".*"

        cursor, err := rethink.
        Table("music").
            Filter(map[string]interface{}{"processed": true}).
            Filter(rethink.Row.Field("title").Match(term)).
            Run(helpers.GetDB())

        helpers.Relax(err)
        defer cursor.Close()

        var results []Song
        err = cursor.All(&results)
        if err == rethink.ErrEmptyResult || len(results) == 0 {
            session.ChannelMessageSend(
                channel.ID,
                "No results :frowning:",
            )
            return
        }
        helpers.Relax(err)

        headers := []string{"Title", "Link"}
        rows := make([][]string, len(results))
        for i, result := range results {
            rows[i] = []string{result.Title, result.URL}
        }

        session.ChannelMessageSend(channel.ID, ":mag: Search results:\n"+helpers.DrawTable(headers, rows))
        break
    }
}

// autoLeave disconnects from VC if the users leave anf forget to !leave
func (m *Module) autoLeave(guildId string, channelId string, session *discordgo.Session, closer <-chan struct{}) {
    defer helpers.Recover()

    for {
        // Wait for 5 seconds
        time.Sleep(5 * time.Second)

        // Exit if the closer channel closes
        select {
        case <-closer:
            return
        default:
        }

        if m.guildConnections[guildId] == nil {
            break
        }

        m.guildConnections[guildId].RLock()
        voiceChannel := session.VoiceConnections[guildId]
        guild, err := session.Guild(guildId)
        if err != nil {
            continue
        }

        fellows := 0
        for _, state := range guild.VoiceStates {
            if state.ChannelID == voiceChannel.ChannelID {
                fellows++
            }
        }

        if fellows == 1 {
            session.ChannelMessageSend(channelId, "Where did everyone go? :frowning:")

            voiceChannel.Disconnect()
            (*m.guildConnections[guild.ID]).CloseChannels()
            delete(m.guildConnections, guild.ID)

            break
        }

        m.guildConnections[guildId].RUnlock()
    }
}

// Waits until the song is ready and notifies.
func (m *Module) waitForSong(channel string, guild string, match Song, msg *discordgo.Message, session *discordgo.Session) {
    defer helpers.RecoverDiscord(msg)

    queue := &m.guildConnections[guild].queue
    playlist := &m.guildConnections[guild].playlist

    for {
        time.Sleep(1 * time.Second)

        cursor, err := rethink.Table("music").Filter(map[string]interface{}{"url": match.URL}).Run(helpers.GetDB())
        helpers.Relax(err)

        var res Song
        err = cursor.One(&res)
        helpers.Relax(err)
        cursor.Close()

        if res.Processed {
            // Remove from queue
            for idx, song := range *queue {
                if song.URL == match.URL {
                    m.guildConnections[guild].Lock()
                    *queue = append((*queue)[:idx], (*queue)[idx+1:]...)
                    m.guildConnections[guild].Unlock()
                }
            }

            // Add to playlist
            m.guildConnections[guild].Lock()
            *playlist = append(*playlist, res)
            m.guildConnections[guild].Unlock()

            session.ChannelMessageSend(channel, ":ballot_box_with_check: `"+res.Title+"` finished downloading :smiley:")
            break
        }
    }
}

// Resolves a voice channel relative to a user id
func (m *Module) resolveVoiceChannel(user *discordgo.User, guild *discordgo.Guild, session *discordgo.Session) *discordgo.Channel {
    for _, vs := range guild.VoiceStates {
        if vs.UserID == user.ID {
            channel, err := session.Channel(vs.ChannelID)
            if err != nil {
                return nil
            }

            return channel
        }
    }

    return nil
}
