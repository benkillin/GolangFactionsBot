package main

import (
	"time"
)

// Config represents the application's configuration
type Config struct {
	Token            string
	TimerLoopTimeout time.Duration
	Logging          LoggingConfig
	Guilds           map[string]*GuildConfig
}

// LoggingConfig configuration as part of the config object.
type LoggingConfig struct {
	Level   string
	Format  string
	Output  string
	Logfile string
}

// GuildConfig represents the configuration of a single instance of this bot on a particular server/guild
type GuildConfig struct {
	GuildName       string
	CommandPrefix   string
	BotRoleAdmin    string
	BotAdminChannel string

	Reminders map[string]*ReminderConfig
	Players   map[string]*PlayerConfig
}

// ReminderConfig represents a type of reminder (Buffers, Walls, Anything else you want to be periodically reminded to check and have the option to weewoo against for the faction to come help with)
type ReminderConfig struct {
	ReminderName     string
	Enabled          bool
	CheckTimeout     time.Duration
	CheckReminder    time.Duration
	LastChecked      time.Time
	CheckChannelID   string
	RoleMention      string
	Reminders        int
	LastReminder     time.Time
	ReminderMessages []string
	WeewoosAllowed   bool   // TODO: add support for this setting
	WeewooMessage    string // TODO: add support for this setting
}

// PlayerConfig represents the players and their scores.
type PlayerConfig struct {
	PlayerString   string
	PlayerUsername string
	PlayerMention  string
	ReminderStats  map[string]*PlayerReminderStats
}

// PlayerReminderStats holds stats for the number of times a particular reminder has been checked and the last time they have checked it.
type PlayerReminderStats struct {
	Weewoos   int
	Checks    int
	LastCheck time.Time
}

// CmdHelp represents a key value pair of a command and a description of a command for constructing a help message embed.
type CmdHelp struct {
	command     string
	description string
}

// TODO: update the below help commands, specifically the set commands, so they provide help for setting reminders of a generic type.

var availableCommands = []CmdHelp{
	CmdHelp{command: "test", description: "A test command."},
	CmdHelp{command: "set", description: "Set settings for the bot such as enabling/disabling wall checks and setting the channel and role for checks."},
	// TODO: might have to require having a specified reminder type for /clear andf /weewoo - or infer based on channel.
	CmdHelp{command: "clear", description: "Mark the walls as all good and clear - nobody is raiding or attacking."},
	CmdHelp{command: "weewoo", description: "Alert fellow faction members that we are getting raided and are under attack!"},
	CmdHelp{command: "help", description: "This help command menu."},
	CmdHelp{command: "invite", description: "Private message you the invite link for this bot to join a server you are an administrator of."},
	CmdHelp{command: "lennyface", description: "Emoji: giggity"},
	CmdHelp{command: "fliptable", description: "Emoji: FLIP THE FREAKING TABLE"},
	CmdHelp{command: "grr", description: "Emoji: i am angry or disappointed with you"},
	CmdHelp{command: "manyface", description: "Emoji: there is nothing but lenny"},
	CmdHelp{command: "finger", description: "Emoji: f you, man"},
	CmdHelp{command: "gimme", description: "Emoji: gimme gimme gimme gimme"},
	CmdHelp{command: "shrug", description: "Emoji: shrug things off"},
}

var setCommands = []CmdHelp{
	CmdHelp{command: "set admin (role)", description: "The role to require to update bot settings on this server (Server administrators always allowed). [admin only]"},
	CmdHelp{command: "set adminChannel (channel)", description: "The channel to put admin related notifications into (such as if someone attempted an operation they are not permitted with the bot). [admin only]"},
	CmdHelp{command: "set prefix (prefix)", description: "Set the command prefix to the specified string. (Defaults to .)."},
	CmdHelp{command: "set addReminder {reminderID}", description: "Add a new reminder type (such as walls, buffers, etc...)."},

	CmdHelp{command: "set reminder {reminderID} reminderName (name)", description: "Set the reminder name of the specified reminder type."},
	CmdHelp{command: "set reminder {reminderID} weewooMsg (message)", description: "Set the message to send if the weewoo command is used."},
	CmdHelp{command: "set reminder {reminderID} weewoo on", description: "Allow weewoos for this reminder type."},
	CmdHelp{command: "set reminder {reminderID} weewoo of", description: "Disable weewoos for this reminder type."},
	CmdHelp{command: "set reminder {reminderID} weewoo cmd {command}", description: "Set the command for a weewoo alert."},
	CmdHelp{command: "set reminder {reminderID} on", description: "Enable checks for specified reminder type."},
	CmdHelp{command: "set reminder {reminderID} off", description: "Disable checks for specified reminder type."},
	CmdHelp{command: "set reminder {reminderID} role (role)", description: "The role to mention for reminders and weewoos, and require for doing clear and weewoo commands (Server administrators always allowed)."},
	CmdHelp{command: "set reminder {reminderID} channel (channel)", description: "The channel to send reminder messages and weewoo alerts to."},
	CmdHelp{command: "set reminder {reminderID} timeout (timeout)", description: "Sets timeout before asking for an action for this reminder. Specify timeout in hours or minutes such as 3m or 2h. Defaults to 45 minutes."},
	CmdHelp{command: "set reminder {reminderID} reminder (reminder)", description: "Sets reminder interval to nag the role for wall checks to check walls. Specify timeout in hours or minutes such as 2m or 1h. Defaults to 30 minutes."},
}
