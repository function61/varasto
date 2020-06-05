package stomediascanner

// below side effects have to be imported to transparently support their decoding

import (
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
)
