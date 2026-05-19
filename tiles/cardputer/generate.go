package cardputerfont

/*
 This is a 6x9 font, specifically designed for the cardputer's 135x240 screen.
 The screen is quite small and the font is accordingly small so a usable number
 of rows and columns are possible. A cell of 6x9 gives exactly 15 rows and 40
 columns with no wasted pixels.

 Happy squinting.
*/

//go:generate go run ../tilegen/main.go -mode=tile-files -pkg=cardputerfont -var=Regular6x9
