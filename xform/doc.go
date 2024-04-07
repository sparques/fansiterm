package xform // github.com/sparques/fansiterm/xform
/*
Package xform has various image transformations for use with both standard lib image.Image and draw.Image.

The transformations work by wrapping an image.Image or a draw.Image. Most of the transformations are nondestructive and work by manipulating what At() returns as At() is called, leaving the underlying image unmodified.


*/
