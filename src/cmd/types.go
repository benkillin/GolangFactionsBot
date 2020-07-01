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
	GuildName           string
	CommandPrefix       string
	BotRoleAdmin        string
	BotAdminChannel     string
	MinimumClearTimeout time.Duration // TODO: add support for configuring this.
	SecretAdmin         string

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
	WeewoosAllowed   bool
	WeewooMessage    string
	WeewooCommand    string
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

// TopStatInfo represents some stats for generating the .top command of top checkers
type TopStatInfo struct {
	playerID string
	stats    PlayerReminderStats
}

var availableCommands = []CmdHelp{
	{command: "test", description: "A test command."},
	{command: "set", description: "Set settings for the bot such as enabling/disabling wall checks and setting the channel and role for checks."},
	// TODO: might have to require having a specified reminder type for /clear andf /weewoo - or infer based on channel.
	{command: "clear", description: "Clear the reminder for checking on whatever this reminder channel is for."},
	//CmdHelp{command: "weewoo", description: "Trigger an alert for whatever this channel is supposed to be a reminder channel for."},
	// TODO: implement the top command
	{command: "top", description: "Display top player statistics. TODO: IMPLEMENT THIS COMMAND"},
	{command: "help", description: "This help command menu."},
	{command: "invite", description: "Private message you the invite link for this bot to join a server you are an administrator of."},
	{command: "lennyface", description: "Emoji: giggity"},
	{command: "fliptable", description: "Emoji: FLIP THE FREAKING TABLE"},
	{command: "grr", description: "Emoji: i am angry or disappointed with you"},
	{command: "manyface", description: "Emoji: there is nothing but lenny"},
	{command: "finger", description: "Emoji: f you, man"},
	{command: "gimme", description: "Emoji: gimme gimme gimme gimme"},
	{command: "shrug", description: "Emoji: shrug things off"},
	// The channel specific weewoo command will be automatically added to the help command if configured.
}

var setCommands = []CmdHelp{
	{command: "set admin (role)", description: "The role to require to update bot settings on this server (Server administrators always allowed). [admin only]"},
	{command: "set adminChannel (channel)", description: "The channel to put admin related notifications into (such as if someone attempted an operation they are not permitted with the bot). [admin only]"},
	{command: "set prefix (prefix)", description: "Set the command prefix to the specified string. (Defaults to .)."},
	// TODO: {command: "set minClearTimeout (timeout)", description: "Set the minimum amount of time between users using the .clear command in a particular channel."},
	{command: "set addReminder {reminderID}", description: "Add a new reminder type (such as walls, buffers, cannon boxes, etc...)."},

	{command: "set reminder", description: "Prints a list of available configured reminders."},
	{command: "set reminder {reminderID} reminderName (name)", description: "Set the reminder name of the specified reminder type."},
	{command: "set reminder {reminderID} weewooMsg (message)", description: "Set the message to send if the weewoo command is used."},
	{command: "set reminder {reminderID} weewooEnabled on", description: "Enable weewoos for this reminder type."},
	{command: "set reminder {reminderID} weewooEnabled off", description: "Disable weewoos for this reminder type."},
	{command: "set reminder {reminderID} weewooCmd {command}", description: "Set the command for a weewoo alert."},
	{command: "set reminder {reminderID} on", description: "Enable checks for specified reminder type."},
	{command: "set reminder {reminderID} off", description: "Disable checks for specified reminder type."},
	{command: "set reminder {reminderID} role (role)", description: "The role to mention for reminders and weewoos, and require for doing clear and weewoo commands (Server administrators always allowed)."},
	{command: "set reminder {reminderID} channel (channel)", description: "The channel to send reminder messages and weewoo alerts to."},
	{command: "set reminder {reminderID} timeout (timeout)", description: "Sets timeout before asking for an action for this reminder. Specify timeout in hours or minutes such as 3m or 2h. Defaults to 45 minutes."},
	{command: "set reminder {reminderID} reminder (reminder)", description: "Sets reminder interval to nag/spam the role for checks for this reminder to clear the reminder. Specify timeout in hours or minutes such as 2m or 1h. Defaults to 30 minutes."},
}
