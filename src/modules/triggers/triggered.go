package triggers

import "code.lukas.moe/x/karen/src/helpers"

type Triggered struct{}

func (t *Triggered) Triggers() []string {
    return []string{
        "triggered",
        "trigger",
    }
}

func (t *Triggered) Response(trigger string, content string) string {
    return helpers.GetText("triggers.triggered")
}
