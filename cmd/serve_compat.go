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
	if tlsClientCA != "" && (tlsCert == "" || tlsKey == "") {
		return fmt.Errorf("tls-client-ca requiere tls-cert y tls-key")
	}

	switch host {
	case "", "127.0.0.1", "localhost", "::1":
		return nil
	}
	if tlsCert == "" || tlsKey == "" {
		return fmt.Errorf("la exposición remota exige tls-cert y tls-key")
	}
	if tlsClientCA == "" {
		return fmt.Errorf("la exposición remota exige tls-client-ca")
	}
	return nil
}

type serverSecurityOptions struct {
	TLSCert     string
	TLSKey      string
	TLSClientCA string
}
