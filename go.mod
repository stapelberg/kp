module github.com/mostfunkyduck/kp

go 1.18

require (
	github.com/AlecAivazis/survey/v2 v2.3.5
	github.com/abiosoft/ishell v2.0.1-0.20190723053747-1b6ad7eb4d5e+incompatible
	github.com/abiosoft/readline v0.0.0-20180607040430-155bce2042db
	github.com/atotto/clipboard v0.1.4
	github.com/sethvargo/go-password v0.2.0
	github.com/tobischo/gokeepasslib/v3 v3.1.0 // cannot be upgraded past v3.1.0 due to a bug in encoding
	zombiezen.com/go/sandpass v1.1.0
)

require (
	github.com/aead/argon2 v0.0.0-20180111183520-a87724528b07 // indirect
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da // indirect
	github.com/chzyer/logex v1.2.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/flynn-archive/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467 // indirect
	golang.org/x/text v0.3.7 // indirect
)

replace github.com/abiosoft/ishell => ./fork/github.com/abiosoft/ishell

replace zombiezen.com/go/sandpass => github.com/mostfunkyduck/sandpass v1.1.1-0.20200617090953-4e7550e75911
