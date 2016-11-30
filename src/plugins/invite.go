package plugins

import (
    "github.com/bwmarrin/discordgo"
    "fmt"
    "../utils"
)

type Invite struct{}

func (i Invite) Name() string {
    return "Invite"
}

func (i Invite) Description() string {
    return "Get an invite link for me"
}

func (i Invite) Commands() map[string]string {
    return map[string]string{
        "invite" : "",
        "inv" : "",
    }
}

func (i Invite) Init(session *discordgo.Session) {

}

func (i Invite) Action(command string, content string, msg *discordgo.Message, session *discordgo.Session) {
    session.ChannelMessageSend(
        msg.ChannelID,
        fmt.Sprintf(
            "To add me to your discord server visit https://discordapp.com/oauth2/authorize?client_id=%s&scope=bot&permissions=%s :smiley:",
            utils.GetConfig().Path("discord.id").Data().(string),
            utils.GetConfig().Path("discord.perms").Data().(string),
        ),
    )
}