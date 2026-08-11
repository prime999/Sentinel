package checker

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func TestVerifyHostnameUsesSANNotOnlyCN(t *testing.T) {
	cert := &x509.Certificate{
		Subject:  pkix.Name{CommonName: "e9d507c3e9.nxcli.io"},
		DNSNames: []string{"e9d507c3e9.nxcli.io", "teqtivity.com", "www.teqtivity.com"},
	}
	if err := cert.VerifyHostname("teqtivity.com"); err != nil {
		t.Fatalf("expected SAN match for teqtivity.com: %v", err)
	}
	if err := cert.VerifyHostname("www.teqtivity.com"); err != nil {
		t.Fatalf("expected SAN match for www.teqtivity.com: %v", err)
	}
	if err := cert.VerifyHostname("other.example"); err == nil {
		t.Fatal("expected hostname mismatch for other.example")
	}
}

func TestSSLChainOKUsesVerifiedChains(t *testing.T) {
	leaf := &x509.Certificate{Subject: pkix.Name{CommonName: "leaf"}}
	state := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{leaf},
		VerifiedChains:   [][]*x509.Certificate{{leaf}},
	}
	if !sslChainOK(leaf, state, "teqtivity.com") {
		t.Fatal("verified chains from handshake should count as OK")
	}
}

func TestSSLChainOKWithIntermediates(t *testing.T) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	rootCert, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}

	interKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	interTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test Intermediate"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	interDER, err := x509.CreateCertificate(rand.Reader, interTmpl, rootCert, &interKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	interCert, err := x509.ParseCertificate(interDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "e9d507c3e9.nxcli.io"},
		DNSNames:     []string{"e9d507c3e9.nxcli.io", "teqtivity.com", "www.teqtivity.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, interCert, &leafKey.PublicKey, interKey)
	if err != nil {
		t.Fatal(err)
	}
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(rootCert)
	opts := x509.VerifyOptions{
		DNSName:       "teqtivity.com",
		Roots:         roots,
		Intermediates: x509.NewCertPool(),
	}
	opts.Intermediates.AddCert(interCert)
	if _, err := leafCert.Verify(opts); err != nil {
		t.Fatalf("SAN + intermediates verify failed: %v", err)
	}

	// Without intermediates this commonly fails — mirrors the old bug.
	badOpts := x509.VerifyOptions{DNSName: "teqtivity.com", Roots: roots}
	if _, err := leafCert.Verify(badOpts); err == nil {
		t.Fatal("expected verify without intermediates to fail")
	}
}
