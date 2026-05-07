//go:generate go run ../tilegen/main.go -mode=font-alpha -fontfile=/usr/share/fonts/TTF/FiraCodeNerdFontMono-Retina.ttf -hinting=full -pkg=firacodenerd -var=Regular8x16 -size=13
//go:generate go run ../tilegen/main.go -mode=font-alpha -fontfile=/usr/share/fonts/TTF/FiraCodeNerdFontMono-Bold.ttf -hinting=full -pkg=firacodenerd -var=Bold8x16 -size=13

package firacodenerd
