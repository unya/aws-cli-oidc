# aws-cli-oidc

CLI tool for retrieving AWS temporary credentials using OIDC provider.


## How does it work?

[AWS Identity Providers and Federation](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers.html) supports IdPs that are compatible with [OpenID Connect (OIDC)](http://openid.net/connect/). This tool works as an OIDC client. If the federation between the AWS account and the IdP is established, and an OIDC client for this tool is registered in the IdP, you can get AWS temporary credentials via standard browser login. It means you don't need to pass your credential of the IdP to this tool.

Please refer the following diagrams how it works.

### Federation type: OIDC

![flow with oidc](flow-with-oidc.png)

## Prerequisite AWS and OIDC provider settings before using this tool

Before using this tool, the system administrator need to setup the following configuration.

- Identity Federation using OIDC between AWS and the OIDC provider. See https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers.html
- Registration OIDC/OAuth2 client for this CLI tool in the OIDC provider. Note: The OIDC provider must allow any port to be specified at the time of the request for loopback IP redirect URIs because this tool follows [RFC 8252 OAuth 2.0 for Native Apps 7.3 Loopback Interface Redirection](https://tools.ietf.org/html/rfc8252#section-7.3).

Also depending on the federation type between AWS and the OIDC provider, requirements for the OIDC providers will change.

### Federation type: OIDC

- The OIDC provider only needs to support OIDC. SAML2 and OAuth 2.0 Token Exchange are not necessary. Very simple.
- However, the JWKS endpoint of the OIDC provider needs to export it to the Internet because AWS try to access the endpoint to obtain the public key and to verify the ID token which is issued by the provider.

## Tested OIDC Provider

- [Google account](https://accounts.google.com/.well-known/openid-configuration)

## Install

Download from [Releases page](https://github.com/mbrtargeting/aws-cli-oidc/releases).


## Usage

```
CLI tool for retrieving AWS temporary credentials using OIDC provider

Usage:
  aws-cli-oidc [command]

Available Commands:
  get-cred    Get AWS credentials and out to stdout
  help        Help about any command
  setup       Interactive setup of aws-cli-oidc

Flags:
  -h, --help   help for aws-cli-oidc

Use "aws-cli-oidc [command] --help" for more information about a command.
```


### Setup

Use `aws-cli-oidc setup` command and follow the guide.


### Get AWS temporary credentials

Use `aws-cli-oidc get-cred <your oidc provider name>` command. 
If you did not log in for a long time or if you are using the tool for the first time, it opens your browser for you to authenticate.
If the authentication is successful, AWS temporary credentials will be output in a JSON format.

You can also use this tool directly as a credential process.
Just add the following lines to your `.aws/credentials` file.
```
[my-profile]
credential_process = aws-cli-oidc get-cred google
```

## Licence

Licensed under the [MIT](/LICENSE) license.


## Authors

- [Hiroyuki Wada](https://github.com/wadahiro)
- [Ströer SSP GmbH](https://www.stroeer.de/konvergenz-konzepte/daten-technologien/stroeer-ssp.html)