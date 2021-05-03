package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/abiosoft/ishell"
	op "github.com/mostfunkyduck/kp/internal/backend/op"
	v1 "github.com/mostfunkyduck/kp/internal/backend/keepassv1"
	v2 "github.com/mostfunkyduck/kp/internal/backend/keepassv2"
	t "github.com/mostfunkyduck/kp/internal/backend/types"
	"github.com/mostfunkyduck/kp/internal/commands"
)

var (
	keyFile        = flag.String("key", "", "a key file to use to unlock the db")
	dbFile         = flag.String("db", "", "the db to open")
	keepassVersion = flag.Int("kpversion", 1, "which version of keepass to use (1 or 2)")
	version        = flag.Bool("version", false, "print version and exit")
	noninteractive = flag.String("n", "", "execute a given command and exit")
)

func fileCompleter(shell *ishell.Shell, printEntries bool) func(string, []string) []string {
	return func(searchPrefix string, searchTargets []string) (ret []string) {
		searchPrefixParts := strings.Split(searchPrefix, "/")

		// if the searchPrefix has a path in it, we need to do our searching in that path for that prefix
		// otherwise, search here, either for the searchTargets, if they exist, or the non-location part of the searchPrefix
		searchLocation := ""
		searchTarget := ""
		if len(searchPrefixParts) > 1 {
			searchLocation = strings.Join(searchPrefixParts[0 : len(searchPrefixParts)-1], "/")
			searchTarget = searchPrefixParts[len(searchPrefixParts)-1]
		}

		// If there's explicit search targets being passed to the tab completion, respect those
		if len(searchTargets) > 0 {
			// if the search target *has* the path, the group name search will fail because it's only the name(title for entries)
			// the search target is split on spaces
			fullSearchTarget := strings.Join(searchTargets, " ")

			// this will split the now fully joined path by slashed
			searchTargetParts := strings.Split(fullSearchTarget, "/")

			// This will make the search target itself be only the name
			searchTarget = searchTargetParts[len(searchTargetParts)-1]

			// unescape the spaces, the shell needs them, but they break search
			searchTarget = strings.ReplaceAll(searchTarget, "\\ ", " ")

			if searchLocation == "" {
				searchLocation = strings.Join(searchTargetParts[0 : len(searchTargetParts)-1], "/")
			}
		}
		// now we find the location object for the current searchPrefix, as parsed above
		db := shell.Get("db").(t.Database)
		location, _, err := commands.TraversePath(db, db.CurrentLocation(), searchLocation)

		// if the there was an error, return nothing
		if err != nil {
			return
		}

		if location == nil {
			location = db.CurrentLocation()
		}

		// There's some duplication here, but basically the idea is that it will search every group and entry
		// in the current location for anything starting with whatever the searchTarget came out to be
		// There appear to be 2 modes in play, one where the user has entered an existing path, in which case the code is supposed to return the full path
		// and one where the user has entered a partial path, in which case the code is supposed to return *only* the completion
		// This drove me crazy. RIP my sanity. Argh.

		returnedPrefix := ""
		doCompletion := searchPrefix == "" || len (searchTargets) >= 1
		if searchLocation != "" {
			returnedPrefix = searchLocation + "/"
		}

		for _, g := range location.Groups() {

			if !strings.HasPrefix(g.Name(), searchTarget) {
				continue
			}

			if searchTarget != "" && doCompletion {
				// we have to return a completion, not a path. don't ask me why
				completedName := strings.Replace(g.Name(), searchTarget, "", 1)
				ret = append(ret, strings.TrimLeft(completedName, " "))
				continue
			}
			ret = append(ret, fmt.Sprintf("%s%s/", returnedPrefix, g.Name()))
		}

		if printEntries {
			for _, e := range location.Entries() {
				if !strings.HasPrefix(e.Title(), searchTarget) {
					continue
				}
				if searchTarget != "" && doCompletion {
					// we have to return a completion, not a path. don't ask me why
					completedTitle := strings.Replace(e.Title(), searchTarget, "", 1)
					ret = append(ret, strings.TrimLeft(completedTitle, " "))
					continue
				}

				ret = append(ret, fmt.Sprintf("%s%s", returnedPrefix, e.Title()))
			}
		}

		return ret
	}
}

func buildVersionString() string {
	return fmt.Sprintf("%s.%s-%s.%s (built on %s from %s)", VersionRelease, VersionBuildDate, VersionBuildTZ, VersionBranch, VersionHostname, VersionRevision)
}

// promptForDBPassword will determine the password based on environment vars or, lacking those, a prompt to the user
func promptForDBPassword(shell *ishell.Shell) (string, error) {
	// we are prompting for the password
	shell.Print("enter database password: ")
	return shell.ReadPasswordErr()
}

// newDB will create or open a DB with the parameters specified.  `open` indicates whether the DB should be opened or not (vs created)
func newDB(dbPath string, password string, keyPath string, version int) (t.Database, error) {
	var dbWrapper t.Database
	switch version {
		case 3:
			dbWrapper = &op.Database{}
		case 2:
			dbWrapper = &v2.Database{}
		case 1:
			dbWrapper = &v1.Database{}
		default:
			return nil, fmt.Errorf("invalid version '%d'", version)
	}
	dbOpts := t.Options {
		DBPath: dbPath,
		Password: password,
		KeyPath: keyPath,
	}
	err := dbWrapper.Init(dbOpts)
	return dbWrapper, err
}

func main() {
	flag.Parse()

	shell := ishell.New()
	if *version {
		shell.Printf("version: %s\n", buildVersionString())
		os.Exit(1)
	}

	var dbWrapper t.Database

	dbPath, exists := os.LookupEnv("KP_DATABASE")
	if !exists {
		dbPath = *dbFile
	}

	// default to the flag argument
	keyPath := *keyFile

	if envKeyfile, found := os.LookupEnv("KP_KEYFILE"); found && keyPath == "" {
		keyPath = envKeyfile
	}

	for {
		// if the password is coming from an environment variable, we need to terminate
		// after the first attempt or it will fall into an infinite loop
		var err error
		password, passwordInEnv := os.LookupEnv("KP_PASSWORD")
		if !passwordInEnv {
			password, err = promptForDBPassword(shell)

			if err != nil {
				shell.Printf("could not retrieve password: %s", err)
				os.Exit(1)
			}
		}

		dbWrapper, err = newDB(dbPath, password, keyPath, *keepassVersion)
		if err != nil {
			// typically, these errors will be a bad password, so we want to keep prompting until the user gives up
			// if, however, the password is in an environment variable, we want to abort immediately so the program doesn't fall
			// in to an infinite loop
			shell.Printf("could not open database: %s\n", err)
			if passwordInEnv {
				os.Exit(1)
			}
			continue
		}
		break
	}

	if dbWrapper.Locked() {
		shell.Printf("Lockfile exists for DB at path '%s', another process is using this database!\n", dbWrapper.SavePath())
		shell.Printf("Open anyways? Data loss may occur. (will only proceed if 'yes' is entered)  ")
		line, err := shell.ReadLineErr()
		if err != nil {
			shell.Printf("could not read user input: %s\n", line)
			os.Exit(1)
		}

		if line != "yes" {
			shell.Println("aborting")
			os.Exit(1)
		}
	}

	if err := dbWrapper.Lock(); err != nil {
		shell.Printf("aborting, could not lock database: %s\n", err)
		os.Exit(1)
	}
	shell.Printf("opened database at %s\n", dbWrapper.SavePath())

	shell.Set("db", dbWrapper)
	shell.SetPrompt(fmt.Sprintf("/%s > ", dbWrapper.CurrentLocation().Name()))

	shell.AddCmd(&ishell.Cmd{
		Name:                "ls",
		Help:                "ls [path]",
		Func:                commands.Ls(shell),
		CompleterWithPrefix: fileCompleter(shell, true),
	})
	shell.AddCmd(&ishell.Cmd{
		Name:                "new",
		Help:                "new <path>",
		LongHelp:            "creates a new entry at <path>",
		Func:                commands.NewEntry(shell),
		CompleterWithPrefix: fileCompleter(shell, false),
	})
	shell.AddCmd(&ishell.Cmd{
		Name:                "mkdir",
		LongHelp:            "create a new group",
		Help:                "mkdir <group name>",
		Func:                commands.NewGroup(shell),
		CompleterWithPrefix: fileCompleter(shell, false),
	})
	shell.AddCmd(&ishell.Cmd{
		Name:     "saveas",
		LongHelp: "save this db to a new file, existing credentials will be used, the new location will /not/ be used as the main save path",
		Help:     "saveas <file.kdb>",
		Func:     commands.SaveAs(shell),
	})

	if dbWrapper.Version() == t.V2 {
		shell.AddCmd(&ishell.Cmd{
			Name:                "select",
			Help:                "select [-f] <entry>",
			LongHelp:            "shows details on a given value in an entry, passwords will be redacted unless '-f' is specified",
			Func:                commands.Select(shell),
			CompleterWithPrefix: fileCompleter(shell, true),
		})
	}

	shell.AddCmd(&ishell.Cmd{
		Name:                "show",
		Help:                "show [-f] <entry>",
		LongHelp:            "shows details on a given entry, passwords will be redacted unless '-f' is specified",
		Func:                commands.Show(shell),
		CompleterWithPrefix: fileCompleter(shell, true),
	})
	shell.AddCmd(&ishell.Cmd{
		Name:                "cd",
		Help:                "cd <path>",
		LongHelp:            "changes the current group to a different path",
		Func:                commands.Cd(shell),
		CompleterWithPrefix: fileCompleter(shell, false),
	})

	attachCmd := &ishell.Cmd{
		Name:     "attach",
		LongHelp: "manages the attachment for a given entry",
		Help:     "attach <get|show|delete> <entry> <filesystem location>",
	}
	attachCmd.AddCmd(&ishell.Cmd{
		Name:                "create",
		Help:                "attach create <entry> <name> <filesystem location>",
		LongHelp:            "creates a new attachment based on a local file",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Attach(shell, "create"),
	})
	attachCmd.AddCmd(&ishell.Cmd{
		Name:                "get",
		Help:                "attach get <entry> <filesystem location>",
		LongHelp:            "retrieves an attachment and outputs it to a filesystem location",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Attach(shell, "get"),
	})
	attachCmd.AddCmd(&ishell.Cmd{
		Name:                "details",
		Help:                "attach details <entry>",
		LongHelp:            "shows the details of the attachment on an entry",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Attach(shell, "details"),
	})
	shell.AddCmd(attachCmd)

	shell.AddCmd(&ishell.Cmd{
		LongHelp:            "searches for any entries with the regular expression '<term>' in their titles or contents",
		Name:                "search",
		Help:                "search <term>",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Search(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:                "rm",
		Help:                "rm <entry>",
		LongHelp:            "removes an entry",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Rm(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:                "xp",
		Help:                "xp <entry>",
		LongHelp:            "copies a password to the clipboard",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Xp(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:                "edit",
		Help:                "edit <entry>",
		LongHelp:            "edits an existing entry",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Edit(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:     "pwd",
		Help:     "pwd",
		LongHelp: "shows path of current group",
		Func:     commands.Pwd(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:     "save",
		Help:     "save",
		LongHelp: "saves the database to its most recently used path",
		Func:     commands.Save(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:     "xx",
		Help:     "xx",
		LongHelp: "clears the clipboard",
		Func:     commands.Xx(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:                "xu",
		Help:                "xu",
		LongHelp:            "copies username to the clipboard",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Xu(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:                "xw",
		Help:                "xw",
		LongHelp:            "copies url to clipboard",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Xw(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:                "mv",
		Help:                "mv <soruce> <destination>",
		LongHelp:            "moves entries between groups",
		CompleterWithPrefix: fileCompleter(shell, true),
		Func:                commands.Mv(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name:     "version",
		Help:     "version",
		LongHelp: "prints version",
		Func: func(c *ishell.Context) {
			shell.Printf("version: %s\n", buildVersionString())
		},
	})

	if *noninteractive != "" {
		bits := strings.Split(*noninteractive, " ")
		if err := shell.Process([]string{bits[0], strings.Join(bits[1:], " ")}...); err != nil {
			shell.Printf("error processing command: %s\n", err)
		}
	} else {
		shell.Run()
	}

	// This will run after the shell exits
	fmt.Println("exiting")

	if dbWrapper.Changed() {
		if err := commands.PromptAndSave(shell); err != nil {
			fmt.Printf("error attempting to save database: %s\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("no changes detected since last save.")
	}


	if err := dbWrapper.Unlock(); err != nil {
		fmt.Printf("failed to unlock db: %s", err)
		os.Exit(1)
	}
}
