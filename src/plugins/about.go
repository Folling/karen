package plugins

import "github.com/bwmarrin/discordgo"

type About struct{}

func (a About) Name() string {
    return "About"
}

func (a About) Description() string {
    return "Shows information about the bot"
}

func (a About) Commands() map[string]string {
    return map[string]string{
        "about" : "",
    }
}

func (a About) Action(command string, content string, msg *discordgo.Message, session *discordgo.Session) {
    m := "Hi my name is Karen!\nI'm a :robot: that will make this Discord Server a better place c:\nHere is some information about me:\n```\n"

    m += `
Karen Araragi (阿良々木 火憐, Araragi Karen) is the eldest of Koyomi Araragi's sisters and the older half of
the Tsuganoki 2nd Middle School Fire Sisters (栂の木二中のファイヤーシスターズ, Tsuganoki Ni-chuu no Faiya Shisutazu).

She is a self-proclaimed "hero of justice" who often imitates the personality and
quirks of various characters from tokusatsu series.
Despite this, she is completely uninvolved with the supernatural, until she becomes victim to a certain oddity.
She is the titular protagonist of two arcs: Karen Bee and Karen Ogre. She is also the narrator of Karen Ogre.
`

    m += "\n```"

    session.ChannelMessageSend(msg.ChannelID, m)
}

func (a About) New() Plugin {
    return &About{}
}