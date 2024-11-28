//go:generate go run -tags gentileset ../gentileset/main.go  -fontfile=https://github.com/kmar/Sweet16Font/raw/refs/heads/master/Sweet16mono.ttf -hinting=none -pkg sweet16 -var Bold8x16 -size 16
//go:generate go run -tags gentileset ../gentileset/main.go  -fontfile=https://github.com/kmar/Sweet16Font/raw/refs/heads/master/Sweet16mono.ttf -hinting=none -pkg sweet16 -var Regular8x16 -size 16 -output=png

package sweet16
