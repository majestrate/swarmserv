package encode

import (
	"encoding/base32"
)

// ZBase32Encoding is the standard lokinet base32 encoding
var ZBase32Encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769").WithPadding(base32.NoPadding)
