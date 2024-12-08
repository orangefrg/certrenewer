package ychelper

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/orangefrg/certrenewer/internal/filehelper"
	"github.com/sirupsen/logrus"
	certmgr "github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"google.golang.org/grpc"
)

type CertificateService interface {
	List(ctx context.Context, req *certmgr.ListCertificatesRequest, opts ...grpc.CallOption) (*certmgr.ListCertificatesResponse, error)
	Get(ctx context.Context, req *certmgr.GetCertificateRequest, opts ...grpc.CallOption) (*certmgr.Certificate, error)
}

type CertificateContentService interface {
	Get(ctx context.Context, req *certmgr.GetCertificateContentRequest, opts ...grpc.CallOption) (*certmgr.GetCertificateContentResponse, error)
}

type CertificateManager struct {
	Certificate        CertificateService
	CertificateContent CertificateContentService
}

type CertConfig struct {
	Name        string `yaml:"name"`
	PrivKeyPath string `yaml:"privKey"`
	ChainPath   string `yaml:"chain"`
	ServiceName string `yaml:"service"`
}

func GetCertificate(folderId string, certName string, dueDate time.Time, cm *CertificateManager) (needsUpdate bool, chain []string, privKey string, err error) {
	certList, err := cm.Certificate.List(context.Background(), &certmgr.ListCertificatesRequest{
		FolderId: folderId,
	})
	if err != nil {
		return false, nil, "", fmt.Errorf("could not list certificates: %w", err)
	}

	certId := ""
	for _, cert := range certList.Certificates {
		if cert.Name == certName {
			certId = cert.Id
			break
		}
	}

	if certId == "" {
		return false, nil, "", errors.New("could not find certificate of given name")
	}

	certInfo, err := cm.Certificate.Get(context.Background(), &certmgr.GetCertificateRequest{
		CertificateId: certId,
		View:          certmgr.CertificateView_BASIC,
	})
	if err != nil {
		return false, nil, "", fmt.Errorf("could not fetch certificate info: %w", err)
	}

	if certInfo.Status != certmgr.Certificate_ISSUED {
		return false, nil, "", fmt.Errorf("could not renew certificate: abnormal cert status %d", int(certInfo.Status))
	}

	if certInfo.NotAfter.AsTime().Equal(dueDate) || certInfo.NotAfter.AsTime().Before(dueDate) {
		return false, nil, "", nil
	}

	certContents, err := cm.CertificateContent.Get(context.Background(), &certmgr.GetCertificateContentRequest{
		CertificateId:    certId,
		PrivateKeyFormat: certmgr.PrivateKeyFormat_PKCS8,
	})
	if err != nil {
		return false, nil, "", fmt.Errorf("could not fetch certificate contents: %w", err)
	}

	return true, certContents.CertificateChain, certContents.PrivateKey, nil
}

func RenewCertificates(folderId string, cm *CertificateManager, certs []CertConfig) (total int, success int) {
	allServicesMap := make(map[string]int)
	for _, singleCert := range certs {
		_, ok := allServicesMap[singleCert.ServiceName]
		if !ok {
			allServicesMap[singleCert.ServiceName] = 0
		}
		allServicesMap[singleCert.ServiceName]++
	}
	total = len(certs)
	var wg sync.WaitGroup
	for index, singleCert := range certs {
		wg.Add(1)
		defer wg.Done()
		go func() {
			logrus.Infof("Updating %s (%d/%d)...", singleCert.Name, index+1, len(certs))
			expiryDate, err := GetCertificateExpiryDate(singleCert.ChainPath)
			if err != nil {
				logrus.Infof("Could not get cert %s expiry date: %s, forcing update", singleCert.Name, err.Error())
				expiryDate = time.Time{}
			}
			needsUpdate, chain, privKey, err := GetCertificate(folderId, singleCert.Name, expiryDate, cm)
			if err != nil {
				logrus.Infof("Could not update cert %s: %s", singleCert.Name, err.Error())
				return
			} else if !needsUpdate {
				logrus.Infof("Certificate %s does not need to be updated", singleCert.Name)
				return
			}
			fullChain := ""
			for _, chainPart := range chain {
				fullChain += chainPart + "\n"
			}
			err = filehelper.WriteWithBackup(singleCert.ChainPath, []byte(fullChain), 0644)
			if err != nil {
				logrus.Infof("Could not write cert %s chain to file %s: %s", singleCert.Name, singleCert.ChainPath, err.Error())
				return
			}
			err = filehelper.WriteWithBackup(singleCert.PrivKeyPath, []byte(privKey), 0600)
			if err != nil {
				logrus.Infof("Could not write cert %s private key to file %s: %s", singleCert.Name, singleCert.PrivKeyPath, err.Error())
				return
			}
			logrus.Infof("Successfully written certificate %s", singleCert.Name)
		}()
	}
	wg.Wait()
	logrus.Info("All certificates updated")
	logrus.Info("Restarting services...")
	for key := range allServicesMap {
		logrus.Infof("Restarting %s", key)
		err := filehelper.ServiceRestart(key)
		if err != nil {
			logrus.Infof("Could not restart %s: %s", key, err.Error())
			continue
		}
		success += allServicesMap[key]
	}
	return
}

var GetCertificateExpiryDate = func(certPath string) (time.Time, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not read certificate file: %w", err)
	}

	var certs []*x509.Certificate

	for {
		var block *pem.Block
		block, certPEM = pem.Decode(certPEM)
		if block == nil {
			break // No more PEM blocks
		}
		if block.Type != "CERTIFICATE" {
			continue // Skip non-certificate PEM blocks
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return time.Time{}, fmt.Errorf("no certificates found in file")
	}

	var leafCert *x509.Certificate
	for _, cert := range certs {
		if !cert.IsCA {
			leafCert = cert
			break
		}
	}

	if leafCert == nil {
		leafCert = certs[0]
	}

	return leafCert.NotAfter, nil
}
