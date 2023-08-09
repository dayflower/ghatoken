package ghatoken

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v53/github"
	"github.com/youmark/pkcs8"
)

type OwnerMode struct {
	TryOrg  bool
	TryUser bool
}

func decodeLegacyPEMasRSAPrivateKey(block *pem.Block, passphrase []byte) (*rsa.PrivateKey, error) {
	if block.Type != "RSA PRIVATE KEY" {
		panic(fmt.Sprintf("Invalid PEM type. '%s'", block.Type))
	}

	if x509.IsEncryptedPEMBlock(block) {
		if len(passphrase) == 0 {
			return nil, fmt.Errorf("encrypted key found, but passphrase was not supplied")
		}

		decrypted, err := x509.DecryptPEMBlock(block, passphrase)
		if err != nil {
			if err == x509.IncorrectPasswordError {
				return nil, fmt.Errorf("incorrect passphrase. %w", err)
			}

			return nil, fmt.Errorf("decrypt failed: %w", err)
		}

		return x509.ParsePKCS1PrivateKey(decrypted)
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func CreateRSAPrivateKeyFromPEM(data []byte, passphrase []byte) (*rsa.PrivateKey, error) {
	block, _ /* rest */ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("PEM block not found")
	}

	var key *rsa.PrivateKey
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err = decodeLegacyPEMasRSAPrivateKey(block, passphrase)
	case "PRIVATE KEY":
		key, err = pkcs8.ParsePKCS8PrivateKeyRSA(block.Bytes)
	case "ENCRYPTED PRIVATE KEY":
		key, err = pkcs8.ParsePKCS8PrivateKeyRSA(block.Bytes, passphrase)
	default:
		return nil, fmt.Errorf("unsupported PEM file: '%s'", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode private key. %w", err)
	}

	return key, nil
}

func NewGitHubClientForApps(appId int64, privateKey *rsa.PrivateKey) (*github.Client, error) {
	itr := ghinstallation.NewAppsTransportFromPrivateKey(http.DefaultTransport, appId, privateKey)

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func NewGitHubEnterpriseClientForApps(appId int64, privateKey *rsa.PrivateKey, baseURL string) (*github.Client, error) {
	itr := ghinstallation.NewAppsTransportFromPrivateKey(http.DefaultTransport, appId, privateKey)
	itr.BaseURL = baseURL

	client, err := github.NewEnterpriseClient(baseURL, baseURL, &http.Client{Transport: itr})
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHubEnterpriseClient. %w", err)
	}

	return client, nil
}

func CreateInstallationTokenForOrg(client *github.Client, org string) (*string, error) {
	return CreateInstallationTokenForOrgOrUser(client, org, OwnerMode{TryOrg: true})
}

func CreateInstallationTokenForUser(client *github.Client, user string) (*string, error) {
	return CreateInstallationTokenForOrgOrUser(client, user, OwnerMode{TryUser: true})
}

func CreateInstallationTokenForOrgOrUser(client *github.Client, owner string, ownerMode OwnerMode) (*string, error) {
	ctx := context.Background()

	var installation *github.Installation

	if ownerMode.TryOrg {
		var resp *github.Response
		var err error
		installation, resp, err = client.Apps.FindOrganizationInstallation(ctx, owner)
		if err != nil {
			if resp.StatusCode == 404 {
				// not found for the org (may retry with user)
			} else {
				return nil, fmt.Errorf("createInstallationTokenForOrgOrUser: failed to find org '%s'. %w", owner, err)
			}
		}
	}

	if ownerMode.TryUser {
		var err error
		installation, _, err = client.Apps.FindUserInstallation(ctx, owner)
		if err != nil {
			return nil, fmt.Errorf("createInstallationTokenForOrgOrUser: failed to find user '%s'. %w", owner, err)
		}
	}

	if installation == nil {
		return nil, fmt.Errorf("createInstallationTokenForOrgOrUser: failed to find org '%s'", owner)
	}

	token, _, err := client.Apps.CreateInstallationToken(ctx, *installation.ID, &github.InstallationTokenOptions{})
	if err != nil {
		return nil, fmt.Errorf("createInstallationTokenForOrgOrUser: failed to create installation token. %w", err)
	}

	return token.Token, nil
}

func CreateInstallationTokenForRepo(client *github.Client, owner string, repo string, restrictScopeToRepo bool) (*string, error) {
	ctx := context.Background()

	installation, _, err := client.Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("createInstallationTokenForRepo: failed to find repo '%s/%s'. %w", owner, repo, err)
	}

	options := &github.InstallationTokenOptions{}

	if restrictScopeToRepo {
		options.Repositories = []string{repo}
	}

	token, _, err := client.Apps.CreateInstallationToken(ctx, *installation.ID, options)
	if err != nil {
		return nil, fmt.Errorf("createInstallationTokenForRepo: failed to create installation token. %w", err)
	}

	return token.Token, nil
}
