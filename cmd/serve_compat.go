/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"
)

func validateServeSecurity(host, tlsCert, tlsKey, tlsClientCA string) error {
	host = strings.TrimSpace(host)
	tlsCert = strings.TrimSpace(tlsCert)
	tlsKey = strings.TrimSpace(tlsKey)
	tlsClientCA = strings.TrimSpace(tlsClientCA)

	if (tlsCert == "") != (tlsKey == "") {
		return fmt.Errorf("tls-cert y tls-key deben indicarse juntos")
	}

	switch host {
	case "", "127.0.0.1", "localhost", "::1":
		return nil
	}
	if tlsClientCA == "" {
		return fmt.Errorf("la exposición remota exige tls-client-ca")
	}
	return nil
}
