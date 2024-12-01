package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/orangefrg/certrenewer/internal/filehelper"
	"github.com/orangefrg/certrenewer/internal/yc_logging"
	"github.com/orangefrg/certrenewer/internal/ychelper"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type MainConfig struct {
	RenewalPeriod   filehelper.Duration   `yaml:"renewalPeriod"`
	HeartBeatPeriod filehelper.Duration   `yaml:"heartBeatPeriod"`
	Certs           []ychelper.CertConfig `yaml:"certs"`
	LogGroup        string                `yaml:"yclogging-group"`
}

func LoadConfig(cfgPath string) (*MainConfig, error) {
	cfgRaw, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}
	var cfg MainConfig
	err = yaml.Unmarshal(cfgRaw, &cfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}
	if cfg.RenewalPeriod.Duration == 0 {
		return nil, fmt.Errorf("renewal period is not set")
	}
	if cfg.HeartBeatPeriod.Duration == 0 {
		cfg.HeartBeatPeriod.Duration = cfg.RenewalPeriod.Duration / 10
		log.Println("Heartbeat period is not set, using 1/10 of renewal period")
	}
	if len(cfg.Certs) == 0 {
		return nil, fmt.Errorf("no certificates in config")
	}
	for index, cert := range cfg.Certs {
		if cert.Name == "" {
			return nil, fmt.Errorf("certificate name is not set for cert %d", index)
		}
		if cert.PrivKeyPath == "" {
			return nil, fmt.Errorf("private key path is not set for cert %d", index)
		}
		if cert.ChainPath == "" {
			return nil, fmt.Errorf("chain path is not set for cert %d", index)
		}
		if cert.ServiceName == "" {
			return nil, fmt.Errorf("service is not set for cert %d", index)
		}
	}
	if cfg.LogGroup == "" {
		cfg.LogGroup = "default"
	}
	return &cfg, nil
}

func HeartbeatWorker(cfg *MainConfig) {
	log.Println("Heartbeat ok")
}

func RenewerWorker(cfg *MainConfig) error {
	log.Println("Initializing SDK")
	sdk, err := ychelper.MakeSDKForInstanceSA()
	if err != nil {
		return fmt.Errorf("could not initialize SDK: %w", err)
	}
	log.Println("Getting VM info")
	vminfo, err := ychelper.GetMeta()
	if err != nil {
		return fmt.Errorf("could not get VM info: %w", err)
	}
	cm := &ychelper.CertificateManager{
		Certificate:        sdk.Certificates().Certificate(),
		CertificateContent: sdk.CertificatesData().CertificateContent(),
	}
	ychelper.RenewCertificates(vminfo.Vendor.FolderId, cm, cfg.Certs)
	log.Println("Done!")
	return nil
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetOutput(os.Stdout)

	log.Println("Starting")
	if len(os.Args) < 2 {
		log.Fatalln("Expecting cfg file path as argument")
	}
	log.Println("Loading config")
	cfgPath := os.Args[1]
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		log.Fatal(fmt.Errorf("could not load config: %w", err))
	}

	ycmeta, err := ychelper.GetMeta()
	if err != nil {
		logrus.Fatal(fmt.Errorf("could not get instance metadata: %w", err))
	}

	hook, err := yc_logging.NewYandexCloudHook(cfg.LogGroup, ycmeta.Vendor.FolderId)
	if err != nil {
		logrus.Fatalf("could not initialize Yandex Cloud hook: %v", err)
	}
	logrus.AddHook(hook)

	stop := make(chan struct{})
	duration := cfg.RenewalPeriod.Duration
	ticker := time.NewTicker(duration)
	defer func() {
		ticker.Stop()
		close(stop)
	}()
	for {
		select {
		case <-ticker.C:
			err := RenewerWorker(cfg)
			if err != nil {
				log.Printf("Error during renewal: %v", err)
			}
		case <-stop:
			log.Println("Stopping")
			return
		}

	}

}
