package messenger

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"strings"

	"github.com/jaytaylor/html2text"
	"github.com/knadh/smtppool"
)

const emName = "email"

// Server represents an SMTP server's credentials.
type Server struct {
	Name          string
	Username      string   `json:"username"`
	Password      string   `json:"password"`
	AuthProtocol  string   `json:"auth_protocol"`
	EmailFormat   string   `json:"email_format"`
	TLSEnabled    bool     `json:"tls_enabled"`
	TLSSkipVerify bool     `json:"tls_skip_verify"`
	InjectHeaders []string `json:"inject_headers"`

	// Rest of the options are embedded directly from the smtppool lib.
	// The JSON tag is for config unmarshal to work.
	smtppool.Opt `json:",squash"`

	pool *smtppool.Pool
}

// Emailer is the SMTP e-mail messenger.
type Emailer struct {
	servers     map[string]*Server
	serverNames []string
	numServers  int
}

// NewEmailer creates and returns an e-mail Messenger backend.
// It takes multiple SMTP configurations.
func NewEmailer(servers ...Server) (*Emailer, error) {
	e := &Emailer{
		servers: make(map[string]*Server),
	}

	for _, srv := range servers {
		s := srv
		var auth smtp.Auth
		switch s.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		case "plain":
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		case "login":
			auth = &smtppool.LoginAuth{Username: s.Username, Password: s.Password}
		case "":
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
		}
		s.Opt.Auth = auth

		// TLS config.
		if s.TLSEnabled {
			s.TLSConfig = &tls.Config{}
			if s.TLSSkipVerify {
				s.TLSConfig.InsecureSkipVerify = s.TLSSkipVerify
			} else {
				s.TLSConfig.ServerName = s.Host
			}
		}

		pool, err := smtppool.New(s.Opt)
		if err != nil {
			return nil, err
		}

		s.pool = pool
		e.servers[s.Name] = &s
		e.serverNames = append(e.serverNames, s.Name)
	}

	e.numServers = len(e.serverNames)
	return e, nil
}

// Name returns the Server's name.
func (e *Emailer) Name() string {
	return emName
}

// Push pushes a message to the server.
func (e *Emailer) Push(fromAddr string, toAddr []string, subject string, m []byte, atts []Attachment) error {
	var key string

	// If there are more than one SMTP servers, send to a random
	// one from the list.
	if e.numServers > 1 {
		key = e.serverNames[rand.Intn(e.numServers)]
	} else {
		key = e.serverNames[0]
	}

	// Are there attachments?
	var files []smtppool.Attachment
	if atts != nil {
		files = make([]smtppool.Attachment, 0, len(atts))
		for _, f := range atts {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	mtext, err := html2text.FromString(string(m), html2text.Options{PrettyTables: true})
	if err != nil {
		return err
	}

	srv := e.servers[key]
	em := smtppool.Email{
		From:        fromAddr,
		To:          toAddr,
		Subject:     subject,
		Attachments: files,
		Headers:     make(map[string][]string),
	}

	for _, header := range srv.InjectHeaders {
		headerParts := strings.SplitN(header, ":", 2)
		// only add the header if both key and value is at least 1 char
		if len(headerParts) == 2 && len(headerParts[0]) > 0 && len(headerParts[1]) > 0 {
			em.Headers.Add(headerParts[0], strings.TrimSpace(headerParts[1]))
		}
	}

	switch srv.EmailFormat {
	case "html":
		em.HTML = m
	case "plain":
		em.Text = []byte(mtext)
	default:
		em.HTML = m
		em.Text = []byte(mtext)
	}

	return srv.pool.Send(em)
}

// Flush flushes the message queue to the server.
func (e *Emailer) Flush() error {
	return nil
}
