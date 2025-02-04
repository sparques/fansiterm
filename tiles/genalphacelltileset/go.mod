module genalphacelltileset

go 1.23

toolchain go1.23.4

require golang.org/x/image v0.15.0

require (
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/sparques/fansiterm v0.0.0-00010101000000-000000000000
)

replace github.com/sparques/fansiterm => /home/sparques/projects/fansiterm
