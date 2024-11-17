package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/orangefrg/certrenewer/internal/ychelper"
	"gopkg.in/yaml.v2"
)

type MainConfig struct {
	RenewalPeriod time.Duration         `yaml:"renewalPeriod"`
	Certs         []ychelper.CertConfig `yaml:"certs"`
}

func main() {
	log.Println("Starting")
	if len(os.Args) < 2 {
		log.Fatalln("Expecting cfg file path as argument")
	}
	log.Println("Loading config")
	cfgPath := os.Args[1]
	cfgRaw, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatal(fmt.Errorf("could not load config: %w", err))
	}
	var cfg MainConfig
	err = yaml.Unmarshal(cfgRaw, &cfg)
	if err != nil {
		log.Fatal(fmt.Errorf("could not parse config: %w", err))
	}
	log.Println("Initializing SDK")
	sdk, err := ychelper.MakeSDK()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Getting VM info")
	vminfo, err := ychelper.GetMeta()
	if err != nil {
		log.Fatal(err.Error())
	}
	cm := &ychelper.CertificateManager{
		Certificate:        sdk.Certificates().Certificate(),
		CertificateContent: sdk.CertificatesData().CertificateContent(),
	}
	ychelper.RenewCertificates(vminfo.Vendor.FolderId, cm, cfg.Certs)
	log.Println("Done!")
}
