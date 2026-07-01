package webhooktls

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func parseCert(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("x509.ParseCertificate() error = %v", err)
	}
	return cert
}

func TestGenerateCertificates(t *testing.T) {
	dnsNames := []string{"gitbot.argocd.svc", "gitbot.argocd.svc.cluster.local"}

	bundle, err := GenerateCertificates(dnsNames)
	if err != nil {
		t.Fatalf("GenerateCertificates() error = %v", err)
	}

	ca := parseCert(t, bundle.CACert)
	srv := parseCert(t, bundle.ServerCert)

	if !ca.IsCA {
		t.Error("CA cert IsCA = false, want true")
	}
	if srv.IsCA {
		t.Error("server cert IsCA = true, want false")
	}

	// The server cert must carry the requested SANs.
	sans := map[string]bool{}
	for _, name := range srv.DNSNames {
		sans[name] = true
	}
	for _, want := range dnsNames {
		if !sans[want] {
			t.Errorf("SAN %q missing from server cert DNSNames %v", want, srv.DNSNames)
		}
	}

	// The server cert must be signed by the CA so the caBundle (ca.crt) validates it.
	if err := srv.CheckSignatureFrom(ca); err != nil {
		t.Errorf("server cert not signed by CA: %v", err)
	}
}
