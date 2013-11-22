package main

type CommandFunc func([]string)

type Command struct {
	name    string
	desc    string
	handler CommandFunc
}

var commands = map[string]Command{}

func RegisterCommand(name string, desc string, handler CommandFunc) {
	commands[name] = Command{name, desc, handler}
}

func RegisterCommands() {
	RC := RegisterCommand
	RC("version", "Show data version information.", VersionCmd)
	RC("help", "Show usage information.", HelpCmd)
	RC("list", "List installed datasets.", ListCmd)
	RC("info", "Show dataset information.", InfoCmd)
}

func PrintCommands() {
	for _, cmd := range commands {
		DErr("    %-10.10s%s\n", cmd.name, cmd.desc)
	}
}

func DispatchCommand(name string, args []string) {

	if DEBUG {
		DErr("dispatching command %s\n", name)
	}

	cmd, ok := commands[name]
	if ok {
		cmd.handler(args)
	} else {
		DErr("data: unknown command \"%s\"\n", name)
		DErr("Run `data help` for usage.\n")
	}
}