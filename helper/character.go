package helper

func IsDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func IsAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func IsAlphaNum(ch byte) bool {
	return IsDigit(ch) || IsAlpha(ch)
}

func IsWhiteSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func IsNewLine(ch byte) bool {
	return ch == '\n'
}

func IsOperator(ch byte) bool {
	switch ch {
	case '+', '-', '*', '/', '%', '=', '<', '>', '!':
		return true
	}
	return false
}

func IsEquality(ch byte) bool {
	switch ch {
	case '=', '>', '<':
		return true
	}
	return false
}
