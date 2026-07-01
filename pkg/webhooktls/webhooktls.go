// Package webhooktls bootstraps the TLS material the admission webhook needs to be reachable
// by the Kubernetes API server: it generates a self-signed CA and a serving certificate, persists
// them in a Secret so every replica shares the same identity, and injects the CA into the
// ValidatingWebhookConfiguration's caBundle.
package webhooktls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Secret data keys. ca.crt is the trust anchor placed manually in the webhook caBundle;
// tls.crt/tls.key are the serving certificate and private key.
const (
	caCertKey     = "ca.crt"
	serverCertKey = "tls.crt"
	serverKeyKey  = "tls.key"
)

// Bundle holds the PEM-encoded CA and server material generated for the webhook.
type Bundle struct {
	CACert     []byte // ca.crt — trust anchor to place in the webhook caBundle
	ServerCert []byte // server.pem — serving certificate (signed by the CA)
	ServerKey  []byte // server.key — serving private key
}

// GenerateCertificates builds a self-signed CA and a server certificate signed by it, valid for
// dnsNames. The CA's ca.crt is what goes into the webhook caBundle; the server cert/key are served.
func GenerateCertificates(dnsNames []string) (Bundle, error) {
	// --- CA ---
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Bundle{}, fmt.Errorf("generate CA key: %w", err)
	}
	caSerial, err := randomSerial()
	if err != nil {
		return Bundle{}, err
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          caSerial,
		Subject:               pkix.Name{CommonName: "gitbot-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return Bundle{}, fmt.Errorf("create CA certificate: %w", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse CA certificate: %w", err)
	}

	// --- Server cert signed by the CA ---
	srvKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return Bundle{}, fmt.Errorf("generate server key: %w", err)
	}
	srvSerial, err := randomSerial()
	if err != nil {
		return Bundle{}, err
	}
	srvTmpl := &x509.Certificate{
		SerialNumber: srvSerial,
		Subject:      pkix.Name{CommonName: dnsNames[0]},
		DNSNames:     dnsNames,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	srvDER, err := x509.CreateCertificate(rand.Reader, srvTmpl, caCert, &srvKey.PublicKey, caKey)
	if err != nil {
		return Bundle{}, fmt.Errorf("create server certificate: %w", err)
	}

	srvKeyDER, err := x509.MarshalECPrivateKey(srvKey)
	if err != nil {
		return Bundle{}, fmt.Errorf("marshal server key: %w", err)
	}

	return Bundle{
		CACert:     pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
		ServerCert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srvDER}),
		ServerKey:  pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: srvKeyDER}),
	}, nil
}

func randomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %w", err)
	}
	return serial, nil
}

// EnsureTLSSecret returns the bundle stored in the named Secret, generating and creating it on
// first use. If a concurrent replica wins the create race (AlreadyExists) it re-reads the winner's
// Secret, so all replicas serve the same certificate and expose the same ca.crt.
func EnsureTLSSecret(ctx context.Context, cs *kubernetes.Clientset, namespace, secretName string,
	dnsNames []string) (Bundle, error) {

	secret, err := cs.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return bundleFromSecret(secret), nil
	}
	if !apierrors.IsNotFound(err) {
		return Bundle{}, fmt.Errorf("get TLS secret %s/%s: %w", namespace, secretName, err)
	}

	// First startup: generate and create the secret.
	bundle, err := GenerateCertificates(dnsNames)
	if err != nil {
		return Bundle{}, err
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			serverCertKey: bundle.ServerCert,
			serverKeyKey:  bundle.ServerKey,
			caCertKey:     bundle.CACert,
		},
	}

	_, err = cs.CoreV1().Secrets(namespace).Create(ctx, newSecret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		// Another replica won the race; use its bundle.
		secret, err = cs.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return Bundle{}, fmt.Errorf("re-read TLS secret after race %s/%s: %w", namespace, secretName, err)
		}
		return bundleFromSecret(secret), nil
	}
	if err != nil {
		return Bundle{}, fmt.Errorf("create TLS secret %s/%s: %w", namespace, secretName, err)
	}

	return bundle, nil
}

func bundleFromSecret(secret *corev1.Secret) Bundle {
	return Bundle{
		CACert:     secret.Data[caCertKey],
		ServerCert: secret.Data[serverCertKey],
		ServerKey:  secret.Data[serverKeyKey],
	}
}

// PatchWebhookCABundle sets clientConfig.caBundle on every webhook of the named
// ValidatingWebhookConfiguration. Idempotent: safe to call on every replica startup.
func PatchWebhookCABundle(ctx context.Context, cs *kubernetes.Clientset, configName string, caBundle []byte) error {
	cfg, err := cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, configName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get ValidatingWebhookConfiguration %s: %w", configName, err)
	}

	for i := range cfg.Webhooks {
		cfg.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	_, err = cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(ctx, cfg, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update ValidatingWebhookConfiguration %s: %w", configName, err)
	}

	return nil
}
