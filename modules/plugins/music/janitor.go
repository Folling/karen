package music

import (
    "strings"
    "os"
    "time"
    "git.lukas.moe/sn0w/Karen/helpers"
    rethink "github.com/gorethink/gorethink"
    "io/ioutil"
    Logger "git.lukas.moe/sn0w/Karen/logger"
)

// janitor watches the data dir and deletes files that don't belong there
func janitor() {
    defer helpers.Recover()

    for {
        // Query for songs
        cursor, err := rethink.Table("music").Run(helpers.GetDB())
        helpers.Relax(err)

        // Get items
        var songs []Song
        err = cursor.All(&songs)
        helpers.Relax(err)
        cursor.Close()

        // If there are no songs continue
        if err == rethink.ErrEmptyResult || len(songs) == 0 {
            continue
        }

        // Remove files that have to DB entry
        dir, err := ioutil.ReadDir("/srv/karen-data")
        helpers.Relax(err)

        for _, file := range dir {
            foundFile := false

            for _, song := range songs {
                if strings.Contains("/srv/karen-data/"+file.Name(), helpers.BtoA(song.URL)) {
                    foundFile = true
                    break
                }
            }

            if !foundFile {
                Logger.INFO.L("music", "[JANITOR] Removing "+file.Name())
                err = os.Remove("/srv/karen-data/" + file.Name())
                helpers.Relax(err)
            }
        }

        time.Sleep(30 * time.Second)
    }
}
