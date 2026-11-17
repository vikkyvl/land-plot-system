package middleware

const (
	headerWWWAuthenticate    = "WWW-Authenticate"
	rejectedRequestChallenge = `Bearer error="invalid_token"`
	formatInsufficientScope  = `Bearer error="insufficient_scope", scope="%s"`
	scopeSeparator           = " "
)
