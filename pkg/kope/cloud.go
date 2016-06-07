package kope

type Cloud interface {
}

type DNSProvider interface {
	ApplyDNSChanges(records map[string][]string) error
}
