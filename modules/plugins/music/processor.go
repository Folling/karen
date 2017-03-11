package music

import (
    "time"
    "os"
    "strconv"
    "os/exec"
    "bufio"
    "io"
    "git.lukas.moe/sn0w/Karen/helpers"
    Logger "git.lukas.moe/sn0w/Karen/logger"
    rethink "github.com/gorethink/gorethink"
)

// processorLoop is a endless coroutine that checks for new songs and spawns youtube-dl as needed
func processorLoop() {
    defer func() {
        helpers.Recover()

        Logger.ERROR.L("music", "The processorLoop died. Please investigate!")
        time.Sleep(5 * time.Second)
        go processorLoop()
    }()

    // Define vars once and override later as needed
    var err error
    var cursor *rethink.Cursor

    for {
        // Sleep before next iteration
        time.Sleep(5 * time.Second)

        // Get unprocessed items
        cursor, err = rethink.Table("music").Filter(map[string]interface{}{"processed": false}).Run(helpers.GetDB())
        helpers.Relax(err)

        // Get song objects
        var songs []Song
        err = cursor.All(&songs)
        helpers.Relax(err)
        cursor.Close()

        // If there are no results skip this iteration
        if err == rethink.ErrEmptyResult || len(songs) == 0 {
            continue
        }

        Logger.INFO.L("music", "Found "+strconv.Itoa(len(songs))+" unprocessed items!")

        // Loop through songs
        for _, song := range songs {
            start := time.Now().Unix()

            name := helpers.BtoA(song.URL)

            Logger.INFO.L("music", "Downloading "+song.URL+" as "+name)

            // Download with youtube-dl
            ytdl := exec.Command(
                "youtube-dl",
                "--abort-on-error",
                "--no-color",
                "--no-playlist",
                "--max-filesize", "1024m",
                "-f", "bestaudio/best[height<=720][fps<=30]/best[height<=720]/[abr<=192]",
                "-x",
                "--audio-format", "wav",
                "--audio-quality", "0",
                "-o", name+".%(ext)s",
                "--exec", "mv {} /srv/karen-data",
                song.URL,
            )
            ytdl.Stdout = os.Stdout
            ytdl.Stderr = os.Stderr
            helpers.Relax(ytdl.Start())
            helpers.Relax(ytdl.Wait())

            // WAV => RAW OPUS
            cstart := time.Now().Unix()
            Logger.INFO.L("music", "PCM => ROPUS | "+name)

            // Create file
            opusFile, err := os.Create("/srv/karen-data/" + name + ".ro")
            helpers.Relax(err)
            writer := bufio.NewWriter(opusFile)

            // Read wav
            cat := exec.Command(
                "ffmpeg",
                "-i", "/srv/karen-data/"+name+".wav",
                "-f", "s16le",
                "-ar", "48000",
                "-ac", "2",
                "-v", "128",
                "pipe:1",
            )

            // Convert wav to raw opus
            ro := exec.Command("ropus")

            // Pipe streams
            r, w := io.Pipe()
            cat.Stdout = w
            ro.Stdin = r
            ro.Stdout = writer

            // Run commands
            helpers.Relax(cat.Start())
            helpers.Relax(ro.Start())

            // Wait until cat loaded the whole file
            helpers.Relax(cat.Wait())
            w.Close()

            // Wait until the file is converted
            helpers.Relax(ro.Wait())
            r.Close()
            opusFile.Close()
            cend := time.Now().Unix()

            // Cleanup
            helpers.Relax(os.Remove("/srv/karen-data/" + name + ".wav"))

            // Mark as processed
            song.Processed = true
            song.Path = "/srv/karen-data/" + name + ".ro"

            // Update db
            _, err = rethink.Table("music").
                Filter(map[string]interface{}{"id": song.ID}).
                Update(song).
                RunWrite(helpers.GetDB())
            helpers.Relax(err)

            end := time.Now().Unix()
            Logger.INFO.L(
                "music",
                "Download took "+strconv.Itoa(int(end-start))+"s "+"| Conversion took "+strconv.Itoa(int(cend-cstart))+"s | File: "+name,
            )
        }
    }
}
