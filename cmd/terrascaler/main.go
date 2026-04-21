package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/containeroo/terrascaler/internal/config"
	"github.com/containeroo/terrascaler/internal/externalgrpc"
	"github.com/containeroo/terrascaler/internal/gitlab"
	"github.com/containeroo/terrascaler/internal/provider"
	"github.com/containeroo/terrascaler/internal/terraform"
)

var Version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	gitlabClient, err := gitlab.New(gitlab.Config{
		BaseURL: cfg.GitLabBaseURL,
		Token:   cfg.GitLabToken,
		Project: cfg.GitLabProject,
		Branch:  cfg.GitLabBranch,
		File:    cfg.FilePath,
		Target: terraform.Target{
			BlockType: cfg.BlockType,
			Labels:    cfg.Labels,
			Attribute: cfg.Attribute,
		},
	}, logger.With("component", "gitlab"))
	if err != nil {
		log.Fatalf("create GitLab client: %v", err)
	}

	listener, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		log.Fatalf("listen on %s: %v", cfg.ListenAddress, err)
	}

	serverOptions, err := grpcServerOptions(cfg)
	if err != nil {
		log.Fatalf("configure gRPC server: %v", err)
	}

	grpcServer := grpc.NewServer(serverOptions...)
	providerServer := provider.New(cfg, gitlabClient, logger.With("component", "provider"))
	externalgrpc.RegisterCloudProviderServer(grpcServer, providerServer)

	logger.Info("starting terrascaler",
		"version", Version,
		"listen_address", cfg.ListenAddress,
		"node_group", cfg.NodeGroupID,
		"min_size", cfg.MinSize,
		"max_size", cfg.MaxSize,
		"gitlab_base_url", cfg.GitLabBaseURL,
		"gitlab_project", cfg.GitLabProject,
		"gitlab_branch", cfg.GitLabBranch,
		"terraform_file", cfg.FilePath,
		"terraform_block_type", cfg.BlockType,
		"terraform_block_labels", cfg.Labels,
		"terraform_attribute", cfg.Attribute,
		"tls_enabled", cfg.TLSCertFile != "",
		"mtls_enabled", cfg.TLSClientCA != "",
	)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("serve gRPC: %v", err)
	}
}

func grpcServerOptions(cfg config.Config) ([]grpc.ServerOption, error) {
	if cfg.TLSCertFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	if cfg.TLSClientCA != "" {
		caPEM, err := os.ReadFile(cfg.TLSClientCA)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("failed to parse client CA")
		}
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = pool
	}

	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsConfig))}, nil
}
