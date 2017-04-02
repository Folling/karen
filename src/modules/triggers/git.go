package triggers

import "git.lukas.moe/sn0w/Karen/x/helpers"

type Git struct{}

func (g *Git) Triggers() []string {
    return []string{
        "git",
        "gitlab",
        "github",
        "repo",
    }
}

func (g *Git) Response(trigger string, content string) string {
    return ":earth_africa: " + helpers.GetText("triggers.git")
}
