package email

import _ "embed"

//go:embed verification_code.html
var VerificationCodeTemplate string
