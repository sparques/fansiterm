module github.com/sparques/fansiterm

go 1.22.2

require golang.org/x/exp v0.0.0-20240409090435-93d18d7e34b8

require github.com/sparques/gfx v0.0.0-20240422165645-8484bc574c25

require (
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/term v0.22.0 // indirect
)

replace (
	github.com/sparques/gfx => /home/sparques/projects/gfx
)