# ghatoken - Issue GitHub Apps installation token

`ghatoken` is a command-line tool designed to issue GitHub Apps installation tokens.
This tool is particularly useful for developers and organizations that utilize GitHub Apps for automating workflows and enhancing project management.

With `ghatoken`, you can specify a private key either as a file or an environment variable,
making it flexible to integrate into various environments.
It supports the use of encrypted private key files, adding an extra layer of security.

#### Background

While there is an excellent product, [github-app-token](https://github.com/tibdex/github-app-token),
designed for generating GitHub App tokens for GitHub Actions,
it primarily serves GitHub Actions use cases.
Unfortunately, it does not support other CI tools like Jenkins or CircleCI.

Recognizing this limitation, we developed ghatoken, a tool that can be used across various CI tools, not just GitHub Actions.
This makes it a more versatile solution for generating GitHub App installation tokens.

## Features

- Allows specification of a private key either as a file or an environment variable.
- Supports the use of an encrypted private key file.
  - In this case, a passphrase for decryption must be specified using an environment variable.
  - Supports PKCS#8 format as well as traditional PKCS#1 format.
- Supports GitHub Enterprise.

## Usage

```
ghatoken [OPTIONS] <App ID> <Repo URL>
```

### Options

- `-f`, `--private-key-file=<Private key file name>` : Specify the private key file.
- `-e`, `--private-key-env=<Private key environment name>` : Specify the environment variable name for the private key.
- `-s`, `--passphrase-env=<Passphrase environment name>` : Specify the environment variable name for the private key passphrase. The default is `PASSPHRASE`.
- `--force-owner=[org|user]` : Force owner recognition. You can specify either `org` or `user`.
- `-r`, `--restrict-scope-repo` : Restrict the token scope to the specified repository only.
- `-h`, `--help` : Show the help message.

You must specify either the `-f` (`--private-key-file`) or `-e` (`--private-key-env`) option.

If you are using an encrypted private key, you must also specify an environment name for the passphrase of the key with `-s` (`--passphrase-env`) option.

### Arguments

- App ID : Specify the GitHub App ID.
- Repo URL : Specify the repository URL.

#### Format of "Repo URL"

The following repo URL formats are supported.

(In the following examples, `dayflower` is used as the organization name and `dayflower/ghatoken` as the repository name)

Even when you specify a repository name as an argument, the installation token will be issued against the whole organization (or user).
If you want to restrict the scope of the token to the repository itself, you must add `-r` or `--restrict-scope-repo` option.

##### Specify repository name (for github.com)

```
https://github.com/dayflower/ghatoken.git
https://github.com/dayflower/ghatoken
git@github.com:dayflower/ghatoken.git
dayflower/ghatoken.git
dayflower/ghatoken
```

##### Specify organization or user name (for github.com)

```
https://github.com/dayflower
git@github.com:dayflower
dayflower
```

##### Specify repository name (for GitHub Enterprise)

(`enterprise.example.net` is used as the domain of GitHub Enterprise)

```
https://enterprise.example.net/dayflower/ghatoken.git
https://enterprise.example.net/dayflower/ghatoken
git@enterprise.example.net:dayflower/ghatoken.git
```

##### Specify organization or user name (for GitHub Enterprise)

```
https://enterprise.example.net/dayflower
git@enterprise.example.net:dayflower
```

## Examples

```bash
$ ghatoken -f encrypted_private.key -s KEY_PASSPHRASE 123 https://github.com/dayflower/
```

This will read an encryped private key from `encrypted_private.key` file and decode the key with an environment value of `KEY_PASSPHRASE`, then issue an installation token for `dayflower` Org (name) of GitHub App (ID 123).

## Guides

### Encrypt RSA private key file

GitHub provides us with a private key for GitHub Apps in a non-encrypted form.
To enhance the security, you can encrypt this key using AES.
Here is a command that uses OpenSSL to achieve this:

```
openssl rsa -in private.key -out encrypted.key -aes256
```

Please replace `private.key` with the path to your actual private key file.
The encrypted key will be saved as `encrypted.key` in the same directory.

When running this command, you will be prompted to enter a passphrase.
This passphrase will be required to use `ghatoken`, so make sure to remember it.

## License

MIT
