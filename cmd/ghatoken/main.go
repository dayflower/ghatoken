package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/dayflower/ghatoken/v0"
	"github.com/google/go-github/v53/github"
	"github.com/jessevdk/go-flags"
)

type repoInfo struct {
	EnterpriseDomain string // if empty, it means this is not enterprise (github.com)
	Owner            string
	Repo             string // if empty, it means target is org or user
}

type appCondition struct {
	AppID               int64
	RepoInfo            *repoInfo
	PrivateKeyInPEM     []byte
	Passphrase          string // if empty, it means passphrase is not specified
	OwnerMode           ghatoken.OwnerMode
	RestrictScopeToRepo bool
}

const githubDotComDomain = "github.com"

/*
Supported repo URL:

	https://github.com/dayflower/ghatoken.git
	https://github.com/dayflower/ghatoken
	git@github.com:dayflower/ghatoken.git
	dayflower/ghatoken.git
	dayflower/ghatoken
	https://github.com/dayflower
	git@github.com:dayflower
	dayflower
	https://enterprise.example.net/dayflower/ghatoken.git
	https://enterprise.example.net/dayflower/ghatoken
	git@enterprise.example.net:dayflower/ghatoken.git
	https://enterprise.example.net/dayflower
	git@enterprise.example.net:dayflower
*/
func parseRepoURL(repoUrl string) (*repoInfo, error) {
	var enterpriseDomain string
	var path string

	if strings.Contains(repoUrl, "://") {
		u, err := url.Parse(repoUrl)
		if err != nil {
			return nil, err
		}

		if u.Hostname() != githubDotComDomain {
			enterpriseDomain = u.Hostname()
		}

		path = u.Path[1:]
	} else if strings.ContainsRune(repoUrl, ':') {
		parts := strings.SplitN(repoUrl, ":", 2+1)
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid repo URL format")
		}

		items := strings.Split(parts[0], "@")
		var domain string
		switch len(items) {
		case 1:
			domain = items[0]
		case 2:
			domain = items[1]
		default:
			return nil, fmt.Errorf("invalid repo URL format")
		}

		if domain != githubDotComDomain {
			enterpriseDomain = domain
		}

		path = parts[1]
	} else {
		path = repoUrl
	}

	items := strings.SplitN(path, "/", 2+1)
	if len(items) > 2 && len(items[2]) > 0 || items[0] == "" {
		return nil, fmt.Errorf("invalid repo URL format")
	}

	var repo string
	if len(items) >= 2 && len(items[1]) > 0 {
		repo, _ = strings.CutSuffix(items[1], ".git")
	}

	return &repoInfo{
		EnterpriseDomain: enterpriseDomain,
		Owner:            items[0],
		Repo:             repo,
	}, nil
}

func apiEndpointForEnterprise(enterpriseDomain string) string {
	return fmt.Sprintf("https://%s/api/v3", enterpriseDomain)
}

type options struct {
	PrivateKeyFile          string `short:"f" long:"private-key-file" description:"Private key file"`
	PrivateKeyEnv           string `short:"e" long:"private-key-env" description:"Private key environment name"`
	PrivateKeyPassphraseEnv string `short:"s" long:"passphrase-env" default:"PASSPHRASE" description:"Private key passphrase environment name"`
	ForceOwner              string `long:"force-owner" choice:"org" choice:"user" description:"Force owner recognition"`
	RestrictScopeToRepo     bool   `short:"r" long:"restrict-scope-repo" description:"Restrict token scope to specified repo only"`

	Version bool `short:"v" long:"version" description:"Show version"`

	Args struct {
		AppID *int64  `description:"GitHub App ID (mandatory)"`
		Repo  *string `description:"Repository URL (mandatory)"`
	} `positional-args:"yes"`
}

func loadPrivateKeyPEM(opts options) (*[]byte, error) {
	var (
		privKey []byte
		err     error
	)

	if len(opts.PrivateKeyFile) > 0 {
		if opts.PrivateKeyFile == "-" {
			privKey, err = io.ReadAll(os.Stdin)
		} else {
			privKey, err = os.ReadFile(opts.PrivateKeyFile)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load private key pem. %w", err)
		}

		return &privKey, nil
	} else if len(opts.PrivateKeyEnv) > 0 {
		privKey = []byte(os.Getenv(opts.PrivateKeyEnv))
		if len(privKey) == 0 {
			return nil, fmt.Errorf("private key environment is not specified")
		}

		return &privKey, nil
	} else {
		return nil, fmt.Errorf("you must specify either file name (--private-key-file) or environment name (--private-key-env) for Private key")
	}
}

var errParseFlags = fmt.Errorf("failed to parse command line arguments")

func parseArgs() (*appCondition, error) {
	var opts options

	_, err := flags.Parse(&opts)
	if err != nil {
		return nil, errParseFlags
	}

	if opts.Version {
		version, revision := GetVersion()
		fmt.Fprintf(os.Stderr, "%s version %s (rev:%s)\n", "ghatoken", version, revision)
		os.Exit(0)
	}

	if opts.Args.AppID == nil {
		return nil, fmt.Errorf("the required arguments `AppID` not provided")
	}
	if opts.Args.Repo == nil {
		return nil, fmt.Errorf("the required arguments `Repo` not provided")
	}

	privKey, err := loadPrivateKeyPEM(opts)
	if err != nil {
		return nil, err
	}

	repoInfo, err := parseRepoURL(*opts.Args.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repo specifier. %w", err)
	}

	var mode ghatoken.OwnerMode
	switch opts.ForceOwner {
	case "":
		mode = ghatoken.OwnerMode{TryOrg: true, TryUser: true}
	case "org":
		mode = ghatoken.OwnerMode{TryOrg: true}
	case "user":
		mode = ghatoken.OwnerMode{TryUser: true}
	default:
		panic("unsupported force-owner mode")
	}

	return &appCondition{
		AppID:               *opts.Args.AppID,
		RepoInfo:            repoInfo,
		PrivateKeyInPEM:     *privKey,
		Passphrase:          os.Getenv(opts.PrivateKeyPassphraseEnv),
		OwnerMode:           mode,
		RestrictScopeToRepo: opts.RestrictScopeToRepo,
	}, nil
}

func handleErrorAndExit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	condition, err := parseArgs()
	if err != nil {
		if err != errParseFlags {
			fmt.Fprintln(os.Stderr, err)
		}

		os.Exit(1)
	}

	key, err := ghatoken.CreateRSAPrivateKeyFromPEM(condition.PrivateKeyInPEM, []byte(condition.Passphrase))
	if err != nil {
		handleErrorAndExit(err)
	}

	var client *github.Client
	if len(condition.RepoInfo.EnterpriseDomain) > 0 {
		client, err = ghatoken.NewGitHubEnterpriseClientForApps(
			condition.AppID,
			key,
			apiEndpointForEnterprise(condition.RepoInfo.EnterpriseDomain),
		)
		if err != nil {
			handleErrorAndExit(err)
		}
	} else {
		client, err = ghatoken.NewGitHubClientForApps(condition.AppID, key)
		if err != nil {
			handleErrorAndExit(err)
		}
	}

	var token *string
	if len(condition.RepoInfo.Repo) == 0 {
		token, err = ghatoken.CreateInstallationTokenForOrgOrUser(client, condition.RepoInfo.Owner, condition.OwnerMode)
		if err != nil {
			handleErrorAndExit(err)
		}
	} else {
		token, err = ghatoken.CreateInstallationTokenForRepo(
			client,
			condition.RepoInfo.Owner,
			condition.RepoInfo.Repo,
			condition.RestrictScopeToRepo,
		)
		if err != nil {
			handleErrorAndExit(err)
		}
	}

	fmt.Print(*token)
}
