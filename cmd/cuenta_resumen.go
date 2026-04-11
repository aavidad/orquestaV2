package cmd

import "strings"

func resumenCuentaCanonica(accountID, email, usuario string) string {
	accountID = strings.TrimSpace(accountID)
	email = strings.TrimSpace(email)
	usuario = strings.TrimSpace(usuario)

	suffix := ""
	if accountID != "" {
		short := accountID
		if len(short) > 8 {
			short = short[:8]
		}
		suffix = " [" + short + "]"
	}

	switch {
	case email != "" && usuario != "" && !strings.EqualFold(email, usuario):
		return email + " (" + usuario + ")" + suffix
	case email != "":
		return email + suffix
	case usuario != "":
		return "usuario " + usuario + suffix
	case accountID != "":
		return "cuenta " + accountID
	default:
		return ""
	}
}
