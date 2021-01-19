package validateemail

import (
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type ValidateEmailError struct {
	err error
}

type ValidateEmailContext struct {
	localhost string
	fromEmail string
}

func (v ValidateEmailError) Error() string {
	return v.err.Error()
}

func (v ValidateEmailError) Code() string {
	return v.err.Error()[:3]
}

func NewValidateEmailError(err error) ValidateEmailError {
	return ValidateEmailError{
		err: err,
	}
}

func NewValidateEmail(localhost, fromEmail string) ValidateEmailContext {
	return ValidateEmailContext{
		localhost: localhost,
		fromEmail: fromEmail,
	}
}

func (v ValidateEmailContext) Validate(email string) error {
	mailSplit := strings.Split(email, "@")
	if (len(mailSplit) != 2) {
		return errors.New("Invalid email format")
	}
	// Procura por servidores que tratam os emails do domínio do email:
	servers := findMailServerFor(mailSplit[1])
	if (len(servers) == 0) {
		return errors.New("Unable to get mailserver from MX records")
	}

	// Tenta se conectar no mailserver do domínio
	// Se a conexão falhar com algum servidor, tenta o próximo até ter sucesso ou
	// acabarem os servidores disponíveis:
	for _, server := range servers {
		conn, err := newSMTPConn(fmt.Sprintf("%s:%d", server, 25))
		if (err != nil) {
			continue;
		}

		err = conn.Hello(v.localhost)
		if (err != nil) {
			return NewValidateEmailError(err);
		}

		err = conn.Mail(v.fromEmail)
		if (err != nil) {
			return NewValidateEmailError(err)
		}

		err = conn.Rcpt(email)
		if (err != nil) {
			return NewValidateEmailError(err)
		}

		return nil
	}
	return errors.New("Unable to connect to mailservers")
}

func findMailServerFor(domain string) []string {
	mxrecords, err := net.LookupMX(domain)
	if (err != nil) {
		return []string{}
	}

	result := make([]string, 0)

	for _, mx := range mxrecords {
		result = append(result, mx.Host)
	}
	return result
}

func newSMTPConn(host string) (*smtp.Client, error) {
	conn, err := net.Dial("tcp", host)
	if (err != nil) {
		return nil, err
	}
	hostOnly, _, _ := net.SplitHostPort(host)

	// Força um timeout para a conexão com o SMTP:
	t := time.AfterFunc(time.Second * 5, func() { conn.Close() })
	defer t.Stop()

	return smtp.NewClient(conn, hostOnly)
}
