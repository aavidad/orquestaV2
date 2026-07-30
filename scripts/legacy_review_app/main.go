// Este fichero compone la herramienta provisional y limita su escucha a loopback.
// No comparte estado con Orquesta ni convierte propuestas en tareas o decisiones.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type configuration struct {
	inventoryPath     string
	proposalsPath     string
	authorizationPath string
	actorRef          string
	projectRef        string
	listen            string
}

func main() {
	catalog := newCatalog("es")
	config, help, err := parseConfiguration(os.Args[1:], catalog, os.Stdout)
	if help {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, catalog.text("cli_start_failed"))
		os.Exit(2)
	}
	if err := runApplication(config, catalog); err != nil {
		fmt.Fprintln(os.Stderr, catalog.text("cli_start_failed"))
		os.Exit(1)
	}
}

func parseConfiguration(arguments []string, catalog catalog, output io.Writer) (configuration, bool, error) {
	var result configuration
	flags := flag.NewFlagSet("legacy_review_app", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&result.inventoryPath, "inventory", "", catalog.text("cli_flag_inventory"))
	flags.StringVar(&result.proposalsPath, "proposals", "", catalog.text("cli_flag_proposals"))
	flags.StringVar(&result.authorizationPath, "authorization-file", "", catalog.text("cli_flag_authorization"))
	flags.StringVar(&result.actorRef, "actor-ref", "", catalog.text("cli_flag_actor"))
	flags.StringVar(&result.projectRef, "project-ref", "", catalog.text("cli_flag_project"))
	flags.StringVar(&result.listen, "listen", "127.0.0.1:8787", catalog.text("cli_flag_listen"))
	help := false
	flags.Usage = func() {
		help = true
		fmt.Fprintln(output, catalog.text("cli_usage"))
	}
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return configuration{}, true, nil
		}
		return configuration{}, false, err
	}
	if help {
		return configuration{}, true, nil
	}
	if flags.NArg() != 0 || strings.TrimSpace(result.inventoryPath) == "" ||
		strings.TrimSpace(result.proposalsPath) == "" || strings.TrimSpace(result.authorizationPath) == "" ||
		!validOpaqueRef(result.actorRef) || !validOpaqueRef(result.projectRef) ||
		samePath(result.inventoryPath, result.proposalsPath) {
		return configuration{}, false, errors.New("configuración inválida")
	}
	if err := validateLoopbackAddress(result.listen); err != nil {
		return configuration{}, false, err
	}
	return result, false, nil
}

func runApplication(config configuration, catalog catalog) error {
	token, err := readAuthorization(config.authorizationPath)
	if err != nil {
		return err
	}
	inventory, err := loadInventory(config.inventoryPath)
	if err != nil {
		return err
	}
	if _, err := readProposals(config.proposalsPath); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", config.listen)
	if err != nil {
		return err
	}
	defer listener.Close()
	host := listener.Addr().String()
	app := &application{
		inventory:     inventory,
		store:         newProposalStore(config.proposalsPath),
		catalog:       catalog,
		authorization: token,
		actorRef:      config.actorRef,
		projectRef:    config.projectRef,
		allowedHost:   host,
		allowedOrigin: "http://" + host,
	}
	server := &http.Server{
		Handler:           app.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 * 1024,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	fmt.Fprintln(os.Stdout, catalog.text("cli_started"), "http://"+host)
	fmt.Fprintln(os.Stdout, catalog.text("cli_login"), "local")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- server.Serve(listener)
	}()
	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals:
		context, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(context); err != nil {
			return err
		}
		<-serveResult
		fmt.Fprintln(os.Stdout, catalog.text("cli_stopped"))
		return nil
	}
}

func validateLoopbackAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("la escucha no es loopback numérico")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 0 || number > 65535 {
		return errors.New("puerto inválido")
	}
	return nil
}

func readAuthorization(filePath string) ([]byte, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("autorización local insegura")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return nil, errors.New("autorización local cambiante")
	}
	content, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return nil, err
	}
	content = []byte(strings.TrimSpace(string(content)))
	if len(content) < 24 || len(content) > 4096 {
		return nil, errors.New("autorización local inválida")
	}
	return content, nil
}

func validOpaqueRef(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 3 && len(value) <= 200 && !strings.ContainsAny(value, "\r\n\t ")
}

func samePath(left, right string) bool {
	leftAbsolute, leftErr := filepath.Abs(left)
	rightAbsolute, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && leftAbsolute == rightAbsolute
}
