package main

import (
    "time"
)

// Config represents the application's configuration
type Config struct {
    Token string
    TimerLoopTimeout time.Duration
    Logging LoggingConfig
    Guilds map[string]*GuildConfig
}

// LoggingConfig configuration as part of the config object.
type LoggingConfig struct {
    Level string
    Format string
    Output string
    Logfile string
}

// GuildConfig represents the configuration of a single instance of this bot on a particular server/guild
type GuildConfig struct {
    GuildName string
    WallsEnabled bool
    WallsCheckTimeout time.Duration
    WallsCheckReminder time.Duration
    WallsLastChecked time.Time
    WallsCheckChannelID string
    WallsRoleMention string
    WallsRoleAdmin string
    WallReminders int
    CommandPrefix string
    ReminderMessages []string
    LastReminder time.Time
    Players map[string]*PlayerConfig
}

// PlayerConfig represents the players and their scores.
type PlayerConfig struct {
    PlayerString string
    PlayerUsername string
    PlayerMention string
    WallChecks int
    LastWallCheck time.Time
}

// CmdHelp represents a key value pair of a command and a description of a command for constructing a help message embed.
type CmdHelp struct {
    command string
    description string
}

var availableCommands = []CmdHelp {CmdHelp {command: "test", description:"A test command."},
        CmdHelp {command: "set", description:"Set settings for the bot such as enabling/disabling wall checks and setting the channel and role for checks."},
        CmdHelp {command: "clear", description:"Mark the walls as all good and clear - nobody is raiding or attacking."},
        CmdHelp {command: "weewoo", description:"Alert fellow faction members that we are getting raided and are under attack!"},
        CmdHelp {command: "help", description:"This help command menu."},
        CmdHelp {command: "invite", description:"Private message you the invite link for this bot to join a server you are an administrator of."},
        CmdHelp {command: "lennyface", description:"Emoji: giggity"},
        CmdHelp {command: "fliptable", description:"Emoji: FLIP THE FREAKING TABLE"},
        CmdHelp {command: "grr", description:"Emoji: i am angry or disappointed with you"},
        CmdHelp {command: "manyface", description:"Emoji: there is nothing but lenny"},
        CmdHelp {command: "finger", description:"Emoji: f you, man"},
        CmdHelp {command: "gimme", description:"Emoji: gimme gimme gimme gimme"},
        CmdHelp {command: "shrug", description:"Emoji: shrug things off"}}

var setCommands = []CmdHelp{CmdHelp {command:"set walls on", description:"Enable wall checks."},
    CmdHelp {command:"set walls off", description:"Disable wall checks."},
    CmdHelp {command: "set walls role (role)", description: "The role to mention for reminders and weewoos, and require for doing clear and weewoo commands (Server administrators always allowed)."},
    CmdHelp {command: "set walls admin (role)", description: "The role to require to update bot settings on this server (Server administrators always allowed)."},
    CmdHelp {command: "set walls channel (channel)", description: "The channel to send reminder messages and weewoo alerts to."},
    CmdHelp {command: "set walls timeout (timeout)", description: "Sets timeout before asking for a wall check. Specify timeout in hours or minutes such as 3m or 2h. Defaults to 45 minutes."},
    CmdHelp {command: "set walls reminder (reminder)", description: "Sets reminder interval to nag the role for wall checks to check walls. Specify timeout in hours or minutes such as 2m or 1h. Defaults to 30 minutes."},
    CmdHelp {command: "set prefix (prefix)", description: "Set the command prefix to the specified string. (Defaults to .)."}}
